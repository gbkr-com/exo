package bitmex

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gbkr-com/exo/env"
	"github.com/gbkr-com/utl"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketOrder(t *testing.T) {

	t.Skip()

	err := env.Load("test.env")
	assert.Nil(t, err)

	errors := []error{}

	conn := &OrderConnection{
		url:    WebSocketTestURL,
		apiKey: os.Getenv("APIKEY"),
		secret: os.Getenv("SECRET"),
		onError: func(e error) {
			errors = append(errors, e)
		},
		limiter:  utl.NewRateLimiter(WebSocketRequestsPerHour, time.Hour),
		lifetime: time.Hour,
	}

	conn.OpenWebSocket()
	<-time.After(3 * time.Second)
	conn.CloseWebSocket()

	// assert.Equal(t, 0, len(errors))
	if len(errors) > 0 {
		fmt.Println(errors[0].Error())
	}

}
