package run

import (
	"context"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gbkr-com/mkt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

func BenchmarkHandler(b *testing.B) {

	//
	// Setup.
	//

	mini := miniredis.RunT(b)
	defer mini.Close()
	rdb := redis.NewClient(&redis.Options{
		Addr: mini.Addr(),
	})

	order := &mkt.Order{
		MsgType: mkt.OrderNew,
		OrderID: mkt.NewOrderID(),
		Side:    mkt.Buy,
		Symbol:  "A",
	}
	quote := &mkt.Quote{
		Symbol:  "A",
		BidPx:   decimal.New(42, 0),
		BidSize: decimal.New(100, 0),
		AskPx:   decimal.New(43, 0),
		AskSize: decimal.New(200, 0),
	}

	ctx, cxl := context.WithCancel(context.Background())
	var shutdown sync.WaitGroup
	completed := make(chan string, 1)

	out := make(chan struct{}, 1)

	proc := NewHandler(
		order,
		&mockDelegateFactory[*mkt.Order]{out: out},
		ConflateTicker,
		rdb,
	)
	shutdown.Add(1)
	go proc.Run(ctx, &shutdown, completed)

	//
	// Benchmark.
	//

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		proc.queue.Push(&Ticker{Quote: quote})
		<-out
	}

	cxl()
	shutdown.Wait()

}
