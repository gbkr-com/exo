package coinbase

import (
	"fmt"
	"testing"
	"time"

	"github.com/gbkr-com/mkt"
	"github.com/stretchr/testify/assert"
)

func TestWebsocket(t *testing.T) {

	t.Skip()

	url := "wss://ws-feed.exchange.coinbase.com"
	errors := []error{}

	conn := &Connection{url: url, symbol: "XRP-USD",
		onQuote: func(q *mkt.Quote) {
			fmt.Println(q)
		},
		onTrade: func(t *mkt.Trade) {
			fmt.Println(t)
		},
		onError: func(e error) {
			errors = append(errors, e)
		},
	}

	conn.Open()
	time.Sleep(2 * time.Second)
	conn.Close()

	assert.Equal(t, 0, len(errors))

}
