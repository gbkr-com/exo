package run

import (
	"context"
	"sync"
	"time"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/redis/go-redis/v9"
)

// NewHandler returns a [*Handler] for an order.
func NewHandler[T mkt.AnyOrder](order T, factory DelegateFactory[T], conflate TickerConflator, rdb *redis.Client) *Handler[T] {

	def := order.Definition()
	// TODO recover last ID's
	return &Handler[T]{
		order:              def,
		queue:              NewTickerConflatingQueue(conflate),
		delegate:           factory.New(order),
		rdb:                rdb,
		instructionsStream: MakeOrderInstructionsStreamName(def),
		reportsStream:      MakeOrderReportsStreamName(def),
		orderHash:          MakeOrderHashKey(def),
	}
}

// A Handler runs for the lifetime of an order, passing ticker data and other
// updates to the [Delegate].
type Handler[T mkt.AnyOrder] struct {
	order              *mkt.Order
	queue              *utl.ConflatingQueue[string, *Ticker]
	delegate           Delegate[T]
	rdb                *redis.Client
	instructionsStream string
	lastInstructionID  string
	reportsStream      string
	lastReportID       string
	orderHash          string
}

// Definition returns the [mkt.Order.Definition] for the [Dispatcher].
func (x *Handler[T]) Definition() *mkt.Order {
	return x.order
}

// Queue returns the queue for the [Dispatcher].
func (x *Handler[T]) Queue() *utl.ConflatingQueue[string, *Ticker] {
	return x.queue
}

// Run until the context is cancelled or the [Delegate] has completed. When
// the [Delegate] completes it sends the OrderID to the given channel, to notify
// the [Dispatcher].
func (x *Handler[T]) Run(ctx context.Context, shutdown *sync.WaitGroup, completed chan<- string) {

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

func (x *Handler[T]) process(ctx context.Context, composite *Ticker, completed chan<- string) (done bool) {
	instructions, reports, err := x.consumeStreams(ctx)
	if err != nil {
		return
	}
	done = x.delegate.Action(composite, instructions, reports)
	x.checkpoint(ctx)
	if done {
		completed <- x.order.OrderID
	}
	return
}

func (x *Handler[T]) consumeStreams(ctx context.Context) (instructions []redis.XMessage, reports []*mkt.Report, err error) {

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
				var report *mkt.Report
				report, err = UnmarshalOrderReport(message)
				if err != nil {
					return
				}
				reports = append(reports, report)
				x.lastReportID = message.ID
			}
		}
	}

	return

}

func (x *Handler[T]) checkpoint(ctx context.Context) error {
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
