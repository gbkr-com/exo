package run

import (
	"context"
	"slices"
	"sync"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// Dispatcher owns all orders and routes information to them.
type Dispatcher[T mkt.AnyOrder] struct {
	instructions chan T
	subscriber   dma.Subscribable
	quotes       *utl.ConflatingQueue[string, *mkt.Quote]
	trades       *utl.ConflatingQueue[string, *mkt.Trade]

	ordersByOrderID map[string]*OrderProcess[T]
	ordersBySymbol  map[string][]*OrderProcess[T]
}

// NewDispatcher returns a [*Dispatcher] ready to use.
func NewDispatcher[T mkt.AnyOrder](
	instructions chan T,
	subscriber dma.Subscribable,
	quotes *utl.ConflatingQueue[string, *mkt.Quote],
	trades *utl.ConflatingQueue[string, *mkt.Trade],
) *Dispatcher[T] {
	dispatcher := &Dispatcher[T]{
		instructions:    instructions,
		subscriber:      subscriber,
		quotes:          quotes,
		trades:          trades,
		ordersByOrderID: make(map[string]*OrderProcess[T]),
		ordersBySymbol:  make(map[string][]*OrderProcess[T]),
	}
	return dispatcher
}

// Run dispatching until the given context is cancelled.
func (x *Dispatcher[T]) Run(ctx context.Context, shutdown *sync.WaitGroup) {

	var processes sync.WaitGroup

	for {

		select {
		case <-ctx.Done():
			//
			// The processes all see the same event through the context.
			//
			processes.Wait()
			shutdown.Done()
			return

		case <-x.quotes.C():
			quote := x.quotes.Pop()
			if quote != nil {
				x.handleQuote(quote)
			}

		case <-x.trades.C():
			trade := x.trades.Pop()
			if trade != nil {
				x.handleTrade(trade)
			}

		case order := <-x.instructions:
			x.handleOrder(ctx, &processes, order)
		}

	}

}

func (x *Dispatcher[T]) handleOrder(ctx context.Context, shutdown *sync.WaitGroup, order T) {

	def := order.Definition()
	process, ok := x.ordersByOrderID[def.OrderID]

	if !ok {
		//
		// Ensure this is correct.
		//
		if def.MsgType != mkt.OrderNew {
			// TODO notification
			return
		}
		//
		// Make a new process for the order.
		//
		process = &OrderProcess[T]{
			queue:        CompositeConflatingQueueFactory[T](),
			instructions: *def,
		}
		x.ordersByOrderID[def.OrderID] = process
		shutdown.Add(1)
		go process.Run(ctx, shutdown)
		//
		// Cross reference by Symbol.
		//
		others, ok := x.ordersBySymbol[def.Symbol]
		if !ok {
			//
			// Subscribe on first appearance.
			//
			x.ordersBySymbol[def.Symbol] = []*OrderProcess[T]{process}
			x.subscriber.Subscribe(def.Symbol)
			return
		}
		x.ordersBySymbol[def.Symbol] = append(others, process)
		return
	}

	if def.MsgType == mkt.OrderNew {
		// TODO notification
		return
	}
	if def.MsgType == mkt.OrderCancel {
		delete(x.ordersByOrderID, def.OrderID)
		others, ok := x.ordersBySymbol[def.Symbol]
		//
		// Unsubscribe here, not in the order process.
		//
		if !ok {
			x.subscriber.Unsubscribe(def.Symbol)
		} else {
			others = slices.DeleteFunc(others, func(p *OrderProcess[T]) bool { return p.instructions.OrderID == def.OrderID })
			if len(others) == 0 {
				x.subscriber.Unsubscribe(def.Symbol)
				delete(x.ordersBySymbol, def.Symbol)
			} else {
				x.ordersBySymbol[def.Symbol] = others
			}
		}
	}
	//
	// Simply push the new order instructions to the queue.
	//
	process.queue.Push(&Composite[T]{Instructions: []T{order}})

}

func (x *Dispatcher[T]) handleQuote(quote *mkt.Quote) {

	processes, ok := x.ordersBySymbol[quote.Symbol]

	if !ok {
		return
	}

	for _, p := range processes {
		composite := &Composite[T]{Quote: quote}
		p.queue.Push(composite)
	}

}

func (x *Dispatcher[T]) handleTrade(trade *mkt.Trade) {

	processes, ok := x.ordersBySymbol[trade.Symbol]

	if !ok {
		return
	}

	for _, p := range processes {
		composite := &Composite[T]{Trade: trade}
		p.queue.Push(composite)
	}

}
