package run

import (
	"context"
	"sync"

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

type entry[T mkt.AnyOrder] struct {
	order T
	quote *mkt.Quote
	trade *mkt.Trade
}

// Dispatcher owns all orders and routes information to them.
type Dispatcher[T mkt.AnyOrder] struct {
	instructions chan T
	quotes       *utl.ConflatingQueue[string, *mkt.Quote]
	trades       *utl.ConflatingQueue[string, *mkt.Trade]

	ordersByOrderID map[string]*entry[T]
	ordersBySymbol  map[string][]*entry[T]
}

// NewDispatcher returns a [*Dispatcher] ready to use.
func NewDispatcher[T mkt.AnyOrder](
	instructions chan T,
	quotes *utl.ConflatingQueue[string, *mkt.Quote],
	trades *utl.ConflatingQueue[string, *mkt.Trade],
) *Dispatcher[T] {
	dispatcher := &Dispatcher[T]{
		instructions:    instructions,
		quotes:          quotes,
		trades:          trades,
		ordersByOrderID: make(map[string]*entry[T]),
		ordersBySymbol:  make(map[string][]*entry[T]),
	}
	return dispatcher
}

// Run dispatching until the given context is cancelled.
func (x *Dispatcher[T]) Run(ctx context.Context, shutdown *sync.WaitGroup) {

	for {

		select {
		case <-ctx.Done():
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
			x.handleOrder(order)
		}

	}

}

func (x *Dispatcher[T]) handleOrder(order T) {

	def := order.Definition()
	_, ok := x.ordersByOrderID[def.OrderID]

	if !ok {
		upd := &entry[T]{order: order}
		x.ordersByOrderID[def.OrderID] = upd
		others, ok := x.ordersBySymbol[def.Symbol]
		if !ok {
			x.ordersBySymbol[def.Symbol] = []*entry[T]{upd}
			// TODO subscribe
			return
		}
		x.ordersBySymbol[def.Symbol] = append(others, upd)
		return
	}

	// TODO

}

func (x *Dispatcher[T]) handleQuote(quote *mkt.Quote) {

	entries, ok := x.ordersBySymbol[quote.Symbol]

	if !ok {
		return
	}

	for _, e := range entries {
		e.quote = quote
	}

}

func (x *Dispatcher[T]) handleTrade(trade *mkt.Trade) {

	entries, ok := x.ordersBySymbol[trade.Symbol]

	if !ok {
		return
	}

	for _, e := range entries {
		if e.trade == nil {
			e.trade = trade
			continue
		}
		if trade.TradeVolume.IsZero() {
			e.trade.Accumulate(trade, 8) // TODO precision
			continue
		}
		t := &mkt.Trade{Symbol: trade.Symbol, LastQty: trade.TradeVolume, LastPx: trade.AvgPx}
		e.trade.Accumulate(t, 8) // TODO precision
	}

}
