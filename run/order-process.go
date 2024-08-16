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

// NewCompositeConflatingQueue makes a composite queue for a single order
// process.
func NewCompositeConflatingQueue[T mkt.AnyOrder]() *utl.ConflatingQueue[string, *Composite[T]] {
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
	queue     *utl.ConflatingQueue[string, *Composite[T]] // Incoming data.
	order     T                                           // The current order instructions.
	completed chan<- string                               // If an order is not cancelled yet completes then the OrderID is sent on this channel.
}

// Run the process until the context is cancelled, a signal that dispatching
// must stop.
func (x *OrderProcess[T]) Run(ctx context.Context, shutdown *sync.WaitGroup) {

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

			x.Action(composite)

		}

	}
}

// Action is where the process responds to events.
func (x *OrderProcess[T]) Action(composite *Composite[T]) {

	if composite.Quote != nil {
		fmt.Println(composite.Quote)
	}

	if composite.Trade != nil {
		fmt.Println(composite.Trade)
	}

}

// Complete signals to the [Dispatcher] that this order process has completed.
func (x *OrderProcess[T]) Complete() {
	x.completed <- x.order.Definition().OrderID
}
