package main

import (
	"fmt"

	"github.com/gbkr-com/exo/run"
	"github.com/gbkr-com/mkt"
)

type delegate struct {
	order *mkt.Order
}

func (x *delegate) Action(upd *run.Composite[*mkt.Order]) bool {

	if upd == nil {
		return false
	}

	if upd.Quote != nil {

		px, size := upd.Quote.Near(x.order.Side)
		fmt.Println(upd.Quote.Symbol, size, px)

	}
	return false
}

func (x *delegate) CleanUp() {}

type delegateFactory struct{}

func (x *delegateFactory) New(order *mkt.Order) run.Delegate[*mkt.Order] {
	return &delegate{order: order}
}
