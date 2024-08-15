package dma

import (
	"fmt"
	"testing"
	"time"

	"github.com/gbkr-com/exo/dma/binance"
	"github.com/gbkr-com/exo/dma/coinbase"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

func TestSubscriber(t *testing.T) {

	t.Skip()

	var (
		c Subscribable
		b Subscribable
	)

	c = NewSubscriber(
		coinbase.WebSocketURL,
		coinbase.Factory,
		func(x *mkt.Quote) { fmt.Println(x) },
		func(x *mkt.Trade) { fmt.Println(x) },
		func(x error) { fmt.Println(x.Error()) },
		utl.NewRateLimiter(coinbase.WebSocketRequestsPerSecond, time.Second),
		time.Hour,
	)

	b = NewSubscriber(
		binance.WebSocketURL,
		binance.Factory,
		func(x *mkt.Quote) { fmt.Println(x) },
		func(x *mkt.Trade) { fmt.Println(x) },
		func(x error) { fmt.Println(x.Error()) },
		utl.NewRateLimiter(binance.WebSocketRequestsPerSecond, time.Second),
		time.Hour,
	)

	b.Subscribe("BTCUSDT")
	c.Subscribe("XRP-USD")

	<-time.After(3 * time.Second)

	b.Unsubscribe("BTCUSDT")
	c.Unsubscribe("XRP-USD")

}
