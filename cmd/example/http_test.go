package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestHTTP(t *testing.T) {
	//
	// Set up.
	//
	mini := miniredis.RunT(t)
	defer mini.Close()
	rdb := redis.NewClient(&redis.Options{
		Addr: mini.Addr(),
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	h := &Handler{
		rdb:          rdb,
		key:          ":hash:orders",
		instructions: make(chan *Order, 16),
	}
	h.Bind(router)

	//
	// POST.
	//
	body := `{
		"side":"BUY",
		"symbol":"XRP-USD",
		"orderQty":"10"
	}
	`
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, basePath, strings.NewReader(body))
	assert.Nil(t, err)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)
	response := struct {
		OrderID string
	}{}
	err = json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err)
	assert.NotEqual(t, "", response.OrderID)

	orderID := response.OrderID

	//
	// Fix up Redis.
	//
	order := <-h.instructions
	assert.NotNil(t, order)
	b, _ := json.Marshal(order)
	rdb.HSet(context.Background(), h.key, order.OrderID, string(b))

	//
	// DELETE.
	//
	w = httptest.NewRecorder()
	req, err = http.NewRequest(http.MethodDelete, basePath+"/"+orderID, nil)
	assert.Nil(t, err)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)

}
