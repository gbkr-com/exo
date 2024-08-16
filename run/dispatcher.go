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
	completedOrders chan string
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
		completedOrders: make(chan string, 1024),
	}
	return dispatcher
}

// Run dispatching until the given context is cancelled. That cancellation is
// a signal that dispatching must stop, not that orders are cancelled.
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

		case orderID := <-x.completedOrders:
			x.removeOrder(orderID)
		}

	}

}

func (x *Dispatcher[T]) handleOrder(ctx context.Context, shutdown *sync.WaitGroup, order T) {

	def := order.Definition()
	process, ok := x.ordersByOrderID[def.OrderID]

	if !ok {
		//
		// A new order - ensure it presents as such.
		//
		if def.MsgType != mkt.OrderNew {
			// TODO notification
			return
		}
		//
		// Make a new process for the order.
		//
		process = &OrderProcess[T]{
			queue:     NewCompositeConflatingQueue[T](),
			order:     order,
			completed: x.completedOrders,
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

	//
	// An existing order, but check first.
	//
	if def.MsgType == mkt.OrderNew {
		// TODO notification
		return
	}
	pdef := process.order.Definition()
	if pdef.Side != def.Side || pdef.Symbol != def.Symbol {
		// TODO notification
		return
	}

	if def.MsgType == mkt.OrderCancel {
		x.removeOrder(def.OrderID)
	}
	//
	// Simply push the new order instructions to the queue.
	//
	process.queue.Push(&Composite[T]{Instructions: []T{order}})

}

func (x *Dispatcher[T]) removeOrder(orderID string) {

	process, ok := x.ordersByOrderID[orderID]
	if !ok {
		return
	}
	symbol := process.order.Definition().Symbol

	delete(x.ordersByOrderID, orderID)

	//
	// Unsubscribe.
	//
	others, ok := x.ordersBySymbol[symbol]
	if !ok {
		x.subscriber.Unsubscribe(symbol)
		return
	}
	others = slices.DeleteFunc(others, func(p *OrderProcess[T]) bool { return p.order.Definition().OrderID == orderID })
	if len(others) == 0 {
		x.subscriber.Unsubscribe(symbol)
		delete(x.ordersBySymbol, symbol)
		return
	}

	x.ordersBySymbol[symbol] = others

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
