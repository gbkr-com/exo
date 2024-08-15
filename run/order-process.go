package run

import (
	"context"
	"fmt"
	"sync"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// Composite is a set of possible updates to be pushed into a [utl.ConflatingQueue]
// for an order process.
type Composite[T mkt.AnyOrder] struct {
	Instructions []T
	Quote        *mkt.Quote
	Trade        *mkt.Trade
}

// CompositeConflatingQueueFactory makes a composite queue for a single order
// process.
func CompositeConflatingQueueFactory[T mkt.AnyOrder]() *utl.ConflatingQueue[string, *Composite[T]] {
	return utl.NewConflatingQueue[string, *Composite[T]](
		func(*Composite[T]) string {
			return ""
		},
		utl.WithConflateOption[string](ConflateComposite[T]),
	)
}

// ConflateComposite conflates updates to a [utl.ConflatingQueue] of [Composite]
// items.
func ConflateComposite[T mkt.AnyOrder](existing *Composite[T], latest *Composite[T]) *Composite[T] {

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

// OrderProcess which is associated with a single OrderID, Symbol and Side.
type OrderProcess[T mkt.AnyOrder] struct {
	queue        *utl.ConflatingQueue[string, *Composite[T]]
	instructions mkt.Order
}

// Run the process until the context is cancelled.
func (x *OrderProcess[T]) Run(ctx context.Context, shutdown *sync.WaitGroup) {

	defer shutdown.Done()

	for {

		select {

		case <-ctx.Done():
			return

		case <-x.queue.C():

			composite := x.queue.Pop()

			if composite.Instructions != nil {
				//
				// Scan for Cancellation.
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

			if composite.Quote != nil {
				fmt.Println(composite.Quote)
			}

			if composite.Trade != nil {
				fmt.Println(composite.Trade)
			}

		}

	}
}
