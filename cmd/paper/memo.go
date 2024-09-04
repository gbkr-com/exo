package main

import (
	"github.com/quickfixgo/enum"
	"github.com/shopspring/decimal"
)

// A Memo for an open order.
type Memo struct {
	OrderID     string
	ClOrdID     string
	Symbol      string
	Side        enum.Side
	OrderQty    decimal.Decimal
	Price       decimal.Decimal
	TimeInForce enum.TimeInForce
}
