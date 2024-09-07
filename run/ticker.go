package run

import (
	"github.com/gbkr-com/exo/env"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// Ticker is a combined update of [mkt.Quote] and [mkt.Trade] to be pushed into
// a [utl.ConflatingQueue]for an [Handler].
type Ticker struct {
	Quote *mkt.Quote
	Trade *mkt.Trade
}

// TickerConflator is any function that can conflate items for the order
// [utl.ConflatingQueue] of [*Ticker] updates.
type TickerConflator func(existing *Ticker, latest *Ticker) *Ticker

// ConflateTicker implements [TickerConflator].
func ConflateTicker(existing *Ticker, latest *Ticker) *Ticker {

	if existing == nil {
		return latest
	}

	if latest.Quote != nil {
		existing.Quote = latest.Quote
	}

	if latest.Trade != nil {
		if existing.Trade == nil {
			existing.Trade = latest.Trade
		} else {
			existing.Trade.Aggregate(latest.Trade, env.DefaultDecimalPlaces)
		}
	}

	return existing
}

// NewTickerConflatingQueue makes a composite queue for an [Handler].
func NewTickerConflatingQueue(fn TickerConflator) *utl.ConflatingQueue[string, *Ticker] {
	return utl.NewConflatingQueue[string, *Ticker](
		func(*Ticker) string {
			return ""
		},
		utl.WithConflateOption[string](fn),
	)
}
