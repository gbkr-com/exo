package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gbkr-com/exo/run"
	"github.com/gbkr-com/mkt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// OrdersHash stores all orders in a Redis hash under this key. Each OrderID is a
// field in the hash.
const OrdersHash = "hash:orders"

// OrdersHashMemo stores the order in the [OrdersHash].
type OrdersHashMemo struct {
	Order
}

// OrderPrefix is the prefix for an OrderID, forming a key of a Redis 'string'
// for each order.
const OrderPrefix = "string:order:"

// OrderMemo is the value associated with an [OrderPrefix] and OrderID.
type OrderMemo struct {
	CumQty decimal.Decimal `json:"cumQty"`
}

// DelegateFactory makes a [Delegate] for each new order.
type DelegateFactory struct {
	rdb *redis.Client
}

// New is called by [run.Dispatcher] when creating the [run.OrderProcess].
func (x *DelegateFactory) New(order *Order) run.Delegate[*Order] {

	if order == nil {
		return nil
	}

	//
	// Persist the delegate when it is created.
	//
	var memo OrdersHashMemo
	memo.Order = *order
	b, err := json.Marshal(memo)
	if err != nil {
		// TODO notify
		return nil
	}
	x.rdb.HSet(context.Background(), OrdersHash, order.OrderID, string(b))

	return &Delegate{
		rdb:   x.rdb,
		order: order,
	}
}

// Delegate is the order handling logic.
type Delegate struct {
	rdb   *redis.Client
	order *Order
	memo  OrderMemo
}

// Action is called by the [run.OrderProcess] for each [run.Ticker] update.
func (x *Delegate) Action(upd *run.Ticker, instructions []redis.XMessage, _ []redis.XMessage) bool {

	if upd == nil {
		return false
	}

	if len(instructions) > 0 {
		//
		// Scan for cancellation.
		//
		for _, ins := range instructions {
			v := ins.Values["msgType"]
			if v == nil {
				continue
			}
			vs := v.(string)
			if vs == mkt.OrderCancel.String() {
				x.removeFromRedis()
				return true
			}
		}
	}

	if upd.Quote != nil {
		//
		// How much is left and how much is available?
		//
		leavesQty := x.order.OrderQty.Sub(x.memo.CumQty)
		px, size := upd.Quote.Far(x.order.Side)
		//
		// Trade (on paper).
		//
		trade := decimal.Min(size, leavesQty)
		fmt.Println(x.order.OrderID, "fill", trade.String(), "@", px.String())
		//
		// Done?
		//
		leavesQty = leavesQty.Sub(trade)
		if leavesQty.IsZero() {
			fmt.Println("done")
			x.removeFromRedis()
			return true
		}
		//
		// Save.
		//
		x.memo.CumQty = x.memo.CumQty.Add(trade)
		x.saveToRedis()

	}

	return false
}

// CleanUp is called by [run.OrderProcess] when [run.Dispatcher] is terminating.
func (x *Delegate) CleanUp() {}

func (x *Delegate) saveToRedis() {
	b, _ := json.Marshal(x.memo)
	x.rdb.Set(context.Background(), OrderPrefix+x.order.OrderID, string(b), 0)
}

func (x *Delegate) removeFromRedis() {
	ctx := context.Background()
	x.rdb.HDel(ctx, OrdersHash, x.order.OrderID)
	x.rdb.Del(ctx, OrderPrefix+x.order.OrderID)
}
