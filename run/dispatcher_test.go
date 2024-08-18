package run

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDispatcherRun(t *testing.T) {

	//
	// Set up.
	//

	ctx, cxl := context.WithCancel(context.Background())
	var shutdown sync.WaitGroup

	instructions := make(chan *mkt.Order, 1)
	reports := make(chan *mkt.Report, 1)

	quoteQueue := utl.NewConflatingQueue(mkt.QuoteKey)
	onQuote := SubscriberQuoteQueueConnector(quoteQueue)
	tradeQueue := utl.NewConflatingQueue(mkt.TradeKey, utl.WithConflateOption[string](ConflateTrade))
	onTrade := SubscriberTradeQueueConnector(tradeQueue)

	subscriber := &mockSubscriber{}

	dispatcher := NewDispatcher(
		instructions,
		&mockDelegateFactory[*mkt.Order]{},
		ConflateComposite,
		reports,
		subscriber,
		quoteQueue,
		tradeQueue,
		func(orderID string, err error) { fmt.Println(orderID, err.Error()) },
	)

	shutdown.Add(1)
	go dispatcher.Run(ctx, &shutdown)

	//
	// Testing.
	//

	orderID := mkt.NewOrderID()

	subscriber.working.Add(1)

	instructions <- &mkt.Order{
		MsgType: mkt.OrderNew,
		OrderID: orderID,
		Side:    mkt.Buy,
		Symbol:  "A",
	}

	subscriber.working.Wait()
	assert.Equal(t, 1, len(subscriber.subs))

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

	subscriber.working.Add(1)

	instructions <- &mkt.Order{
		MsgType: mkt.OrderCancel,
		OrderID: orderID,
		Side:    mkt.Buy,
		Symbol:  "A",
	}

	subscriber.working.Wait()
	assert.Equal(t, 0, len(subscriber.subs))

	cxl()
	shutdown.Wait()

}
