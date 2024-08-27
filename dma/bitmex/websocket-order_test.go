package bitmex

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/exo/env"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketOrder(t *testing.T) {

	t.Skip()

	err := env.Load("test.env")
	assert.Nil(t, err)
	APIKEY := os.Getenv("APIKEY")
	assert.NotEqual(t, "", APIKEY)
	SECRET := os.Getenv("SECRET")
	assert.NotEqual(t, "", SECRET)

	const SYMBOL = "XBTUSD"

	errors := []error{}
	var quote *mkt.Quote

	//
	// Get a quote for the order.
	//
	conn := &Connection{
		url:    WebSocketTestURL,
		symbol: SYMBOL,
		onQuote: func(q *mkt.Quote) {
			quote = q
		},
		onTrade: func(t *mkt.Trade) {},
		onError: func(e error) {
			errors = append(errors, e)
		},
		limiter:  utl.NewRateLimiter(WebSocketRequestsPerHour, time.Hour),
		lifetime: time.Hour,
	}
	conn.OpenWebSocket()
	<-time.After(2 * time.Second)
	conn.CloseWebSocket()
	assert.Equal(t, 0, len(errors))
	assert.NotNil(t, quote)

	fmt.Println(quote)

	//
	// Listen for order and execution messages.
	//
	exec := &OrderConnection{
		url:    WebSocketTestURL,
		apiKey: APIKEY,
		secret: SECRET,
		onError: func(e error) {
			errors = append(errors, e)
		},
		limiter:  utl.NewRateLimiter(WebSocketRequestsPerHour, time.Hour),
		lifetime: time.Hour,
	}
	exec.OpenWebSocket()

	//
	// Make an order.
	//
	open := &dma.OpenOrder{
		Side:        mkt.Buy,
		Symbol:      SYMBOL,
		OrderQty:    decimal.New(1, -6),
		Price:       quote.AskPx,
		TimeInForce: mkt.IOC,
	}
	nr := open.MakeNewRequest()
	no, err := NewOrder(nr, OrderTestURL, APIKEY, SECRET)
	assert.Nil(t, err)

	client := &http.Client{}
	resp, err := client.Do(no)
	assert.Nil(t, err)

	fmt.Println(resp)

	<-time.After(3 * time.Second)
	exec.CloseWebSocket()

	// assert.Equal(t, 0, len(errors))
	if len(errors) > 0 {
		fmt.Println(errors[0].Error())
	}

}
