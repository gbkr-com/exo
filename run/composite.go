package run

import (
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// Composite is a set of independent updates to be pushed into a [utl.ConflatingQueue]
// for an [OrderProcess].
type Composite[T mkt.AnyOrder] struct {
	Instructions []T
	Reports      []*mkt.Report
	Quote        *mkt.Quote
	Trade        *mkt.Trade
}

// CompositeConflator is any function that can conflate items for the order
// [utl.ConflatingQueue] of [*Composite] updates.
type CompositeConflator[T mkt.AnyOrder] func(existing *Composite[T], latest *Composite[T]) *Composite[T]

// ConflateComposite implements [CompositeConflator].
func ConflateComposite[T mkt.AnyOrder](existing *Composite[T], latest *Composite[T]) *Composite[T] {

	if existing == nil {
		return latest
	}

	if len(latest.Instructions) > 0 {
		existing.Instructions = append(existing.Instructions, latest.Instructions...)
	}

	if len(latest.Reports) > 0 {
		existing.Reports = append(existing.Reports, latest.Reports...)
	}

	if latest.Quote != nil {
		existing.Quote = latest.Quote
	}

	if latest.Trade != nil {
		if existing.Trade == nil {
			existing.Trade = latest.Trade
		} else {
			existing.Trade.Aggregate(latest.Trade, 8) // TODO precision
		}
	}

	return existing
}

// NewCompositeConflatingQueue makes a composite queue for an [OrderProcess].
func NewCompositeConflatingQueue[T mkt.AnyOrder](fn CompositeConflator[T]) *utl.ConflatingQueue[string, *Composite[T]] {
	return utl.NewConflatingQueue[string, *Composite[T]](
		func(*Composite[T]) string {
			return ""
		},
		utl.WithConflateOption[string](fn),
	)
}
