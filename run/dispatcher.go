package run

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/redis/go-redis/v9"
)

// Dispatcher owns all orders and routes information to them.
type Dispatcher[T mkt.AnyOrder] struct {
	instructions chan T
	factory      DelegateFactory[T]
	conflator    TickerConflator
	reports      chan *mkt.Report
	subscriber   dma.Subscribable
	quotes       *utl.ConflatingQueue[string, *mkt.Quote]
	trades       *utl.ConflatingQueue[string, *mkt.Trade]
	onError      func(string, error)
	rdb          *redis.Client

	ordersByOrderID map[string]*OrderProcess[T]
	ordersBySymbol  map[string][]*OrderProcess[T]
	completedOrders chan string
}

// NewDispatcher returns a [*Dispatcher] ready to use.
func NewDispatcher[T mkt.AnyOrder](
	instructions chan T,
	factory DelegateFactory[T],
	conflator TickerConflator,
	reports chan *mkt.Report,
	subscriber dma.Subscribable,
	quotes *utl.ConflatingQueue[string, *mkt.Quote],
	trades *utl.ConflatingQueue[string, *mkt.Trade],
	onError func(string, error),
	rdb *redis.Client,
) *Dispatcher[T] {
	dispatcher := &Dispatcher[T]{
		instructions:    instructions,
		factory:         factory,
		conflator:       conflator,
		reports:         reports,
		subscriber:      subscriber,
		quotes:          quotes,
		trades:          trades,
		ordersByOrderID: make(map[string]*OrderProcess[T]),
		ordersBySymbol:  make(map[string][]*OrderProcess[T]),
		completedOrders: make(chan string, 1024), // TODO configure
		onError:         onError,
		rdb:             rdb,
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

		case orderID := <-x.completedOrders:
			x.removeOrder(orderID)

		case order := <-x.instructions:
			x.handleOrder(ctx, &processes, order)

		case report := <-x.reports:
			x.handleReport(report)

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
			x.onError(def.OrderID, fmt.Errorf("Dispatcher: expected mkt.OrderNew, received %s", def.MsgType.String()))
			return
		}
		//
		// Make a new process for the order.
		//
		process = NewOrderProcess(order, x.factory, x.conflator, x.rdb)
		x.ordersByOrderID[def.OrderID] = process
		shutdown.Add(1)
		go process.Run(ctx, shutdown, x.completedOrders)
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
	// An existing order, but check it matches first.
	//
	if def.MsgType == mkt.OrderNew {
		x.onError(def.OrderID, fmt.Errorf("Dispatcher: unexpected mkt.OrderNew"))
		return
	}
	pdef := process.Definition()
	if pdef.Side != def.Side || pdef.Symbol != def.Symbol {
		x.onError(def.OrderID, fmt.Errorf("Dispatcher: Side or Symbol do not match"))
		return
	}

	if def.MsgType == mkt.OrderCancel {
		x.removeOrder(def.OrderID)
	}
	//
	// Add the new order instructions to the order specific Redis stream. Use a
	// different context to ensure there is no interference between the parent
	// being cancelled and the Redis operations completing.
	//
	if err := WriteOrderInstructions(context.Background(), x.rdb, order); err != nil {
		x.onError(def.OrderID, fmt.Errorf("Dispatcher: cannot write order to stream: %w", err))
	}

}

func (x *Dispatcher[T]) removeOrder(orderID string) {

	process, ok := x.ordersByOrderID[orderID]
	if !ok {
		return
	}
	symbol := process.Definition().Symbol

	delete(x.ordersByOrderID, orderID)

	//
	// Unsubscribe.
	//
	others, ok := x.ordersBySymbol[symbol]
	if !ok {
		x.subscriber.Unsubscribe(symbol)
		return
	}
	others = slices.DeleteFunc(others, func(p *OrderProcess[T]) bool { return p.Definition().OrderID == orderID })
	if len(others) == 0 {
		x.subscriber.Unsubscribe(symbol)
		delete(x.ordersBySymbol, symbol)
		return
	}

	x.ordersBySymbol[symbol] = others

}

func (x *Dispatcher[T]) handleReport(report *mkt.Report) {

	if _, ok := x.ordersByOrderID[report.OrderID]; !ok {
		x.onError(report.OrderID, fmt.Errorf("Dispatcher: unexpected mkt.Report"))
		return
	}
	if err := WriteOrderReport(context.Background(), x.rdb, report); err != nil {
		x.onError(report.OrderID, fmt.Errorf("Dispatcher: cannot write report to stream"))
	}

}

func (x *Dispatcher[T]) handleQuote(quote *mkt.Quote) {

	processes, ok := x.ordersBySymbol[quote.Symbol]

	if !ok {
		return
	}

	for _, p := range processes {
		composite := &Ticker{Quote: quote}
		p.Queue().Push(composite)
	}

}

func (x *Dispatcher[T]) handleTrade(trade *mkt.Trade) {

	processes, ok := x.ordersBySymbol[trade.Symbol]

	if !ok {
		return
	}

	for _, p := range processes {
		composite := &Ticker{Trade: trade}
		p.Queue().Push(composite)
	}

}
