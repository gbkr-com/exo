package run

import (
	"context"
	"sync"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// NewOrderProcess returns an [*OrderProcess] for a new order.
func NewOrderProcess[T mkt.AnyOrder](order T, factory DelegateFactory[T], conflate CompositeConflator[T]) *OrderProcess[T] {
	return &OrderProcess[T]{
		order:    order,
		queue:    NewCompositeConflatingQueue[T](conflate),
		delegate: factory.New(order),
	}
}

// An OrderProcess runs for the lifetime of an order.
type OrderProcess[T mkt.AnyOrder] struct {
	order    T                                           // The current order instructions.
	queue    *utl.ConflatingQueue[string, *Composite[T]] // The composite queue given to this process.
	delegate Delegate[T]
}

// Definition returns the [mkt.Order.Definition] for the [Dispatcher].
func (x *OrderProcess[T]) Definition() *mkt.Order {
	return x.order.Definition()
}

// Queue returns the queue for the [Dispatcher].
func (x *OrderProcess[T]) Queue() *utl.ConflatingQueue[string, *Composite[T]] {
	return x.queue
}

// Run until the context is cancelled or the [Delegate] has completed.
func (x *OrderProcess[T]) Run(ctx context.Context, shutdown *sync.WaitGroup, completed chan<- string) {

	defer shutdown.Done()

	for {

		select {

		case <-ctx.Done():
			x.delegate.CleanUp()
			return

		case <-x.queue.C():
			composite := x.queue.Pop()
			if x.delegate.Action(composite) {
				completed <- x.order.Definition().OrderID
				return
			}

		}

	}
}
