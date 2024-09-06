package run

import (
	"context"
	"sync"
	"time"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/redis/go-redis/v9"
)

// NewOrderProcess returns an [*OrderProcess] for an order.
func NewOrderProcess[T mkt.AnyOrder](order T, factory DelegateFactory[T], conflate TickerConflator, rdb *redis.Client) *OrderProcess[T] {
	// TODO recover last ID's
	return &OrderProcess[T]{
		order:              order,
		queue:              NewTickerConflatingQueue(conflate),
		delegate:           factory.New(order),
		rdb:                rdb,
		instructionsStream: MakeOrderInstructionsStreamName(order.Definition()),
		reportsStream:      MakeOrderReportsStreamName(order.Definition()),
		orderHash:          MakeOrderHashKey(order.Definition()),
	}
}

// An OrderProcess runs for the lifetime of an order.
type OrderProcess[T mkt.AnyOrder] struct {
	order              T                                     // The current order instructions.
	queue              *utl.ConflatingQueue[string, *Ticker] // The composite queue given to this process.
	delegate           Delegate[T]
	rdb                *redis.Client
	instructionsStream string
	lastInstructionID  string
	reportsStream      string
	lastReportID       string
	orderHash          string
}

// Definition returns the [mkt.Order.Definition] for the [Dispatcher].
func (x *OrderProcess[T]) Definition() *mkt.Order {
	return x.order.Definition()
}

// Queue returns the queue for the [Dispatcher].
func (x *OrderProcess[T]) Queue() *utl.ConflatingQueue[string, *Ticker] {
	return x.queue
}

// Run until the context is cancelled or the [Delegate] has completed. When
// the [Delegate] completes it sends the OrderID to the given channel, to notify
// the [Dispatcher].
func (x *OrderProcess[T]) Run(ctx context.Context, shutdown *sync.WaitGroup, completed chan<- string) {

	defer shutdown.Done()

	for {

		select {

		case <-ctx.Done():
			x.delegate.CleanUp()
			return

		case <-x.queue.C():
			ticker := x.queue.Pop()
			if x.process(ctx, ticker, completed) {
				return
			}

		case <-time.After(time.Second): // TODO configure
			//
			// For slow trading listings, look for instructions and/or
			// reports when there is no ticker update for some time.
			//
			if x.process(ctx, nil, completed) {
				return
			}

		}

	}
}

func (x *OrderProcess[T]) process(ctx context.Context, composite *Ticker, completed chan<- string) (done bool) {
	instructions, reports, err := x.consumeStreams(ctx)
	if err != nil {
		return
	}
	done = x.delegate.Action(composite, instructions, reports)
	x.checkpoint(ctx)
	if done {
		completed <- x.order.Definition().OrderID
	}
	return
}

func (x *OrderProcess[T]) consumeStreams(ctx context.Context) (instructions []redis.XMessage, reports []redis.XMessage, err error) {

	args := &redis.XReadArgs{
		Streams: []string{x.instructionsStream, x.reportsStream, x.lastInstructionID, x.lastReportID},
		Count:   0,
		Block:   time.Millisecond,
	}

	var streams []redis.XStream
	if streams, err = x.rdb.XRead(ctx, args).Result(); err != nil {
		return
	}

	for _, stream := range streams {
		switch stream.Stream {
		case x.instructionsStream:
			for _, message := range stream.Messages {
				instructions = append(instructions, message)
				x.lastInstructionID = message.ID
			}
		case x.reportsStream:
			for _, message := range stream.Messages {
				reports = append(reports, message)
				x.lastReportID = message.ID
			}
		}
	}

	return

}

func (x *OrderProcess[T]) checkpoint(ctx context.Context) error {
	_, err := x.rdb.HSet(
		ctx,
		x.orderHash,
		x.instructionsStream,
		x.lastInstructionID,
		x.reportsStream,
		x.lastReportID,
	).Result()
	return err
}
