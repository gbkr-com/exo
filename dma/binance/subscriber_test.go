package binance

import (
	"fmt"
	"testing"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

func TestSubscriber(t *testing.T) {

	t.Skip()

	var b dma.Subscribable

	b = dma.NewSubscriber(
		WebSocketURL,
		Factory,
		func(x *mkt.Quote) { fmt.Println(x) },
		func(x *mkt.Trade) { fmt.Println(x) },
		func(x error) { fmt.Println(x.Error()) },
		utl.NewRateLimiter(WebSocketRequestsPerSecond, time.Second),
		time.Hour,
	)

	b.Subscribe("BTCUSDT")

	<-time.After(3 * time.Second)

	b.Unsubscribe("BTCUSDT")

}
