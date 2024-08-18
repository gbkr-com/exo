package main

import (
	"github.com/gbkr-com/mkt"
	"github.com/shopspring/decimal"
)

// An Order defined for this example.
type Order struct {
	mkt.Order
	OrderQty *decimal.Decimal `json:"orderQty"` // FIX field 38, must be present for OrderNew.
}
