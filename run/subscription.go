package run

import (
	"github.com/gbkr-com/exo/env"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// SubscriberQuoteQueueConnector provides the 'onQuote' callback function for
// a [dma.Subscriber].
func SubscriberQuoteQueueConnector(queue *utl.ConflatingQueue[string, *mkt.Quote]) func(*mkt.Quote) {
	return func(quote *mkt.Quote) {
		if quote == nil {
			return
		}
		queue.Push(quote)
	}
}

// SubscriberTradeQueueConnector provides the 'onTrade' callback function for
// a [dma.Subscriber].
func SubscriberTradeQueueConnector(queue *utl.ConflatingQueue[string, *mkt.Trade]) func(*mkt.Trade) {
	return func(trade *mkt.Trade) {
		if trade == nil {
			return
		}
		queue.Push(trade)
	}
}

// ConflateTrade is a convenience function for a trade [utl.ConflatingQueue].
func ConflateTrade(existing *mkt.Trade, latest *mkt.Trade) *mkt.Trade {

	if existing == nil {
		return latest
	}

	existing.Aggregate(latest, env.DefaultDecimalPlaces)
	return existing
}
