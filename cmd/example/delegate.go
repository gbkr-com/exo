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

// DelegateMemo is the data that is persisted for every order.
type DelegateMemo struct {
	Instructions struct {
		Order
	} `json:"instructions"`
	State struct {
		CumQty decimal.Decimal `json:"cumQty"`
	}
}

// DelegateFactory makes a [Delegate] for each new order.
type DelegateFactory struct {
	rdb *redis.Client
	key string
}

// New is called by [run.Dispatcher] when creating the [run.OrderProcess].
func (x *DelegateFactory) New(order *Order) run.Delegate[*Order] {

	if order == nil {
		return nil
	}

	//
	// Persist the delegate when it is created.
	//
	var memo DelegateMemo
	memo.Instructions.Order = *order
	b, err := json.Marshal(memo)
	if err != nil {
		// TODO notify
		return nil
	}
	x.rdb.HSet(context.Background(), x.key, order.OrderID, string(b))

	return &Delegate{
		rdb:  x.rdb,
		key:  x.key,
		memo: &memo,
	}
}

// Delegate is the order handling logic.
type Delegate struct {
	rdb  *redis.Client
	key  string
	memo *DelegateMemo
}

// Action is called by the [run.OrderProcess] for each [run.Composite] update.
func (x *Delegate) Action(upd *run.Composite[*Order]) bool {

	if upd == nil {
		return false
	}

	if len(upd.Instructions) > 0 {
		//
		// Scan for cancellation.
		//
		for _, ins := range upd.Instructions {
			if ins.MsgType == mkt.OrderCancel {
				x.removeFromRedis()
				return true
			}
		}
	}

	if upd.Quote != nil {
		//
		// How much is left and how much is available?
		//
		leavesQty := x.memo.Instructions.OrderQty.Sub(x.memo.State.CumQty)
		px, size := upd.Quote.Far(x.memo.Instructions.Side)
		//
		// Trade (on paper).
		//
		trade := decimal.Min(size, leavesQty)
		fmt.Println("trade", trade.String(), "@", px.String())
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
		x.memo.State.CumQty = x.memo.State.CumQty.Add(trade)
		x.saveToRedis()

	}

	return false
}

// CleanUp is called by [run.OrderProcess] when [run.Dispatcher] is terminating.
func (x *Delegate) CleanUp() {}

func (x *Delegate) saveToRedis() {
	b, _ := json.Marshal(x.memo)
	x.rdb.HSet(context.Background(), x.key, x.memo.Instructions.OrderID, string(b))
}

func (x *Delegate) removeFromRedis() {
	x.rdb.HDel(context.Background(), x.key, x.memo.Instructions.OrderID)
}
