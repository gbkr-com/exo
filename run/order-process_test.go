package run

import (
	"context"
	"sync"
	"testing"

	"github.com/gbkr-com/mkt"
	"github.com/shopspring/decimal"
)

func BenchmarkOrderProcess(b *testing.B) {

	//
	// Setup.
	//

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

	proc := NewOrderProcess(
		order,
		&mockDelegateFactory[*mkt.Order]{out: out},
		ConflateComposite,
	)
	shutdown.Add(1)
	go proc.Run(ctx, &shutdown, completed)

	//
	// Benchmark.
	//

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		proc.queue.Push(&Composite[*mkt.Order]{Quote: quote})
		<-out
	}

	cxl()
	shutdown.Wait()

}
