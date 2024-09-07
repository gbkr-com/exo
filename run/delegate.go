package run

import (
	"github.com/gbkr-com/mkt"
	"github.com/redis/go-redis/v9"
)

// Delegate is the interface for a [Handler] to delegate work on an order.
// A [Delegate] contains all the logic for reacting to ticker data and execution
// reports.
type Delegate[T mkt.AnyOrder] interface {
	// Action the [*Ticker], instruction and report updates. Return true if
	// the order is now complete and no further action is necessary.
	//
	// Each instruction should transform into an instance of T, which is only
	// known by the [Delegate], not the [OrderProcess].
	Action(ticker *Ticker, instructions []redis.XMessage, reports []*mkt.Report) bool
	// CleanUp instructs the [Delegate] to prepare for the [Dispatcher] to exit.
	// This does not mean the order is cancelled.
	CleanUp()
}

// DelegateFactory is used by [Dispatcher] to manufacture a [Delegate] for a
// new order.
type DelegateFactory[T mkt.AnyOrder] interface {
	New(order T) Delegate[T]
}
