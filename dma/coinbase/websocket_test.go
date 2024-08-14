package coinbase

import (
	"fmt"
	"testing"
	"time"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/stretchr/testify/assert"
)

func TestWebsocket(t *testing.T) {

	t.Skip()

	errors := []error{}

	conn := &Connection{
		url:    WebSocketURL,
		symbol: "XRP-USD",
		onQuote: func(q *mkt.Quote) {
			fmt.Println(q)
		},
		onTrade: func(t *mkt.Trade) {
			fmt.Println(t)
		},
		onError: func(e error) {
			errors = append(errors, e)
		},
		limiter:  utl.NewRateLimiter(WebSocketRequestsPerSecond, time.Second),
		lifetime: 2 * time.Second,
	}

	conn.Open()
	<-time.After(4 * time.Second)
	conn.Close()

	assert.Equal(t, 0, len(errors))

}
