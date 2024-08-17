package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gbkr-com/exo/run"
	"github.com/gbkr-com/mkt"
	"github.com/redis/go-redis/v9"
)

type delegateFactory struct {
	rdb *redis.Client
	key string
}

func (x *delegateFactory) New(order *mkt.Order) run.Delegate[*mkt.Order] {

	if order == nil {
		return nil
	}
	b, err := json.Marshal(order)
	if err != nil {
		// TODO notify
		return nil
	}
	x.rdb.HSet(context.Background(), x.key, order.OrderID, string(b))

	return &delegate{
		rdb:   x.rdb,
		key:   x.key,
		order: order,
	}
}

type delegate struct {
	rdb   *redis.Client
	key   string
	order *mkt.Order
}

func (x *delegate) Action(upd *run.Composite[*mkt.Order]) bool {

	if upd == nil {
		return false
	}

	if len(upd.Instructions) > 0 {
		for _, ins := range upd.Instructions {
			if ins.MsgType == mkt.OrderCancel {
				//
				// Redis.
				//
				x.rdb.HDel(context.Background(), x.key, x.order.OrderID)
				return true
			}
		}
	}

	if upd.Quote != nil {

		px, size := upd.Quote.Near(x.order.Side)
		fmt.Println(upd.Quote.Symbol, size, px)

	}
	return false
}

func (x *delegate) CleanUp() {}
