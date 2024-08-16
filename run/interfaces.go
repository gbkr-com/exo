package run

import (
	"context"
	"sync"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// CompositeConflator is any function that can conflate items for the order
// [utl.ConflatingQueue].
type CompositeConflator[T mkt.AnyOrder] func(existing *Composite[T], latest *Composite[T]) *Composite[T]

// OrderProcessorFactory makes an [OrderProcessor].
type OrderProcessorFactory[T mkt.AnyOrder] func(order T) OrderProcessor[T]

// An OrderProcessor runs for the lifetime of an order. The OrderID, Side and
// Symbol are immutable once the process is launched.
type OrderProcessor[T mkt.AnyOrder] interface {
	Definition() mkt.Order
	Queue() *utl.ConflatingQueue[string, *Composite[T]]
	Run(context.Context, *sync.WaitGroup, chan<- string)
}
