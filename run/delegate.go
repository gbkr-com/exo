package run

import (
	"github.com/gbkr-com/mkt"
)

// Delegate is the interface for an [OrderProcess] to delegate work on an
// order.
type Delegate[T mkt.AnyOrder] interface {
	// Action the [*Composite] update. Return true if the order is now complete
	// and no further action is necessary.
	Action(*Composite[T]) bool
	// CleanUp instructs the [Delegate] to prepare for the [Dispatcher] to exit.
	// This does not mean the order is cancelled.
	CleanUp()
}

// DelegateFactory is used by [Dispatcher] to manufacture a [Delegate] for
// new orders.
type DelegateFactory[T mkt.AnyOrder] interface {
	New(order T) Delegate[T]
}
