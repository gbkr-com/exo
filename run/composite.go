package run

import (
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// Composite is a set of possible updates to be pushed into a [utl.ConflatingQueue]
// for an [OrderProcessor].
type Composite[T mkt.AnyOrder] struct {
	Instructions []T
	Quote        *mkt.Quote
	Trade        *mkt.Trade
}

// NewCompositeConflatingQueue makes a composite queue for an [OrderProcessor].
func NewCompositeConflatingQueue[T mkt.AnyOrder](fn CompositeConflator[T]) *utl.ConflatingQueue[string, *Composite[T]] {
	return utl.NewConflatingQueue[string, *Composite[T]](
		func(*Composite[T]) string {
			return ""
		},
		utl.WithConflateOption[string](fn),
	)
}
