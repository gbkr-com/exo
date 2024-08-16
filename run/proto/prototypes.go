package proto

import (
	"context"
	"fmt"
	"sync"

	"github.com/gbkr-com/exo/run"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// ConflateComposite conflates updates to a [utl.ConflatingQueue] of [run.Composite]
// items. It implements [run.CompositeConflator].
func ConflateComposite[T mkt.AnyOrder](existing *run.Composite[T], latest *run.Composite[T]) *run.Composite[T] {

	if existing == nil {
		return latest
	}

	if len(latest.Instructions) > 0 {
		existing.Instructions = append(existing.Instructions, latest.Instructions...)
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

// NewOrderProcess implements [run.OrderProcessorFactory].
func NewOrderProcess[T mkt.AnyOrder](order T) run.OrderProcessor[T] {
	return &OrderProcess[T]{order: order, queue: run.NewCompositeConflatingQueue[T](ConflateComposite)}
}

// OrderProcess which is associated with a single OrderID, Symbol and Side.
type OrderProcess[T mkt.AnyOrder] struct {
	order T                                               // The current order instructions.
	queue *utl.ConflatingQueue[string, *run.Composite[T]] // The composite queue given to this process.
}

// Definition implements [run.OrderProcessor].
func (x *OrderProcess[T]) Definition() mkt.Order { return *x.order.Definition() }

// Queue implements [run.OrderProcessor].
func (x *OrderProcess[T]) Queue() *utl.ConflatingQueue[string, *run.Composite[T]] { return x.queue }

// Run the process until the context is cancelled, a signal that dispatching
// must stop.
// Run implements [run.OrderProcessor].
func (x *OrderProcess[T]) Run(ctx context.Context, shutdown *sync.WaitGroup, completed chan<- string) {

	defer shutdown.Done()

	for {

		select {

		case <-ctx.Done():
			// TODO clean up
			return

		case <-x.queue.C():

			composite := x.queue.Pop()

			if composite.Instructions != nil {
				//
				// Scan for cancellation.
				//
				var cancelled bool
				for _, ins := range composite.Instructions {
					def := ins.Definition()
					if def.MsgType == mkt.OrderCancel {
						cancelled = true
						break
					}
				}
				if cancelled {
					// TOD0 cleanup
					return
				}
			}

			if x.action(composite) {
				completed <- x.order.Definition().OrderID
				// TODO clean up
				return
			}

		}

	}
}

// action is where the process responds to events. This returns true if the
// order is complete.
func (x *OrderProcess[T]) action(composite *run.Composite[T]) bool {

	if composite.Quote != nil {
		fmt.Println(composite.Quote)
	}

	if composite.Trade != nil {
		fmt.Println(composite.Trade)
	}

	return false

}
