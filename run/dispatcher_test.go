package run

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/shopspring/decimal"
)

func TestDispatcherRun(*testing.T) {

	ctx, cxl := context.WithCancel(context.Background())
	var shutdown sync.WaitGroup

	instructions := make(chan *mkt.Order, 1)

	quoteQueue := utl.NewConflatingQueue[string, *mkt.Quote](mkt.QuoteKey)
	onQuote := SubscriberQuoteQueueConnector(quoteQueue)
	tradeQueue := utl.NewConflatingQueue[string, *mkt.Trade](mkt.TradeKey, utl.WithConflateOption[string, *mkt.Trade](ConflateTrade))
	onTrade := SubscriberTradeQueueConnector(tradeQueue)

	dispatcher := NewDispatcher(instructions, quoteQueue, tradeQueue)

	shutdown.Add(1)
	go dispatcher.Run(ctx, &shutdown)

	instructions <- &mkt.Order{
		OrderID: mkt.NewOrderID(),
		Side:    mkt.Buy,
		Symbol:  "A",
	}

	onQuote(
		&mkt.Quote{
			Symbol:  "A",
			BidPx:   decimal.New(42, 0),
			BidSize: decimal.New(100, 0),
			AskPx:   decimal.New(43, 0),
			AskSize: decimal.New(150, 0),
		},
	)
	onTrade(
		&mkt.Trade{
			Symbol:  "A",
			LastQty: decimal.New(10, 0),
			LastPx:  decimal.New(43, 0),
		},
	)

	<-time.After(3 * time.Second)
	cxl()
	shutdown.Wait()

}
