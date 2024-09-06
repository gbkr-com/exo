package run

import (
	"github.com/gbkr-com/mkt"
	"github.com/redis/go-redis/v9"
)

// Delegate is the interface for an [OrderProcess] to delegate work on an
// order.
type Delegate[T mkt.AnyOrder] interface {
	// Action the [*Composite], instruction and report updates. Return true if
	// the order is now complete and no further action is necessary.
	Action(ticker *Ticker, instructions []redis.XMessage, reports []redis.XMessage) bool
	// CleanUp instructs the [Delegate] to prepare for the [Dispatcher] to exit.
	// This does not mean the order is cancelled.
	CleanUp()
}

// DelegateFactory is used by [Dispatcher] to manufacture a [Delegate] for
// new orders.
type DelegateFactory[T mkt.AnyOrder] interface {
	New(order T) Delegate[T]
}
