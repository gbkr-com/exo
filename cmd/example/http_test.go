package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gbkr-com/mkt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHTTP(t *testing.T) {
	//
	// Set up.
	//
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	h := &Handler{
		orders:       map[string]*mkt.Order{},
		instructions: make(chan *mkt.Order, 16),
	}
	h.Bind(router)

	//
	// Test.
	//
	body := `{
		"side":"BUY",
		"symbol":"XRP-USD"
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

	assert.Equal(t, 1, len(h.orders))
	order := h.orders[orderID]
	assert.NotNil(t, order)

	w = httptest.NewRecorder()
	req, err = http.NewRequest(http.MethodDelete, basePath+"/"+orderID, nil)
	assert.Nil(t, err)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)

}
