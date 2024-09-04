package dma

import (
	"github.com/gbkr-com/mkt"
	"github.com/shopspring/decimal"
)

// An OpenOrder is the current state of an order sent to the counterparty. The
// [OpenOrder.OrderID] is that from the [mkt.AnyOrder.Definition]. A single
// [OpenOrder.OrderID] may create many [OpenOrder]. The counterparty generated
// ID for each [OpenOrder] is the [OpenOrder.SecondaryOrderID].
type OpenOrder struct {
	Account          string          // FIX field 1
	OrderID          string          // FIX field 37
	SecondaryOrderID string          // FIX field 198
	ClOrdID          string          // FIX field 11
	Side             mkt.Side        // FIX field 54
	Symbol           string          // FIX field 55
	OrderQty         decimal.Decimal // FIX field 38
	Price            decimal.Decimal // FIX field 44
	TimeInForce      mkt.TimeInForce // FIX field 59
	PendingNew       *NewRequest
	PendingReplace   *ReplaceRequest
	PendingCancel    *CancelRequest
	Complete         bool // TODO
}

// NewOpenOrder returns an [*OpenOrder] from the given [mkt.AnyOrder.Definition].
func NewOpenOrder(def *mkt.Order) *OpenOrder {
	return &OpenOrder{
		OrderID: def.OrderID,
		Side:    def.Side,
		Symbol:  def.Symbol,
	}
}

// IsPending returns true if there is an outstanding request.
func (x *OpenOrder) IsPending() bool {
	return x.PendingNew != nil || x.PendingReplace != nil || x.PendingCancel != nil
}

// MakeNewRequest returns a [*NewRequest] if the state allows.
func (x *OpenOrder) MakeNewRequest() *NewRequest {
	if x.Complete {
		return nil
	}
	if x.IsPending() {
		return nil
	}
	if x.SecondaryOrderID != "" {
		return nil
	}
	request := &NewRequest{
		OpenOrder:   x,
		ClOrdID:     mkt.NewOrderID(),
		Side:        x.Side,
		Symbol:      x.Symbol,
		OrderQty:    x.OrderQty,
		Price:       x.Price,
		TimeInForce: x.TimeInForce,
	}
	x.ClOrdID = request.ClOrdID
	x.PendingNew = request
	return request
}

// MakeReplaceRequest returns a [*ReplaceRequest] if the state allows.
func (x *OpenOrder) MakeReplaceRequest(orderQty *decimal.Decimal, price *decimal.Decimal) *ReplaceRequest {
	if x.Complete {
		return nil
	}
	if x.IsPending() {
		return nil
	}
	if x.SecondaryOrderID == "" {
		return nil
	}
	if orderQty == nil && price == nil {
		return nil
	}
	request := &ReplaceRequest{
		OpenOrder:   x,
		ClOrdID:     mkt.NewOrderID(),
		OrigClOrdID: x.ClOrdID,
		OrderQty:    orderQty,
		Price:       price,
	}
	x.PendingReplace = request
	return request
}

// MakeCancelRequest returns a [*CancelRequest] if the state allows.
func (x *OpenOrder) MakeCancelRequest() *CancelRequest {
	if x.Complete {
		return nil
	}
	if x.IsPending() {
		return nil
	}
	if x.SecondaryOrderID == "" {
		return nil
	}
	request := &CancelRequest{
		OpenOrder:   x,
		ClOrdID:     mkt.NewOrderID(),
		OrigClOrdID: x.ClOrdID,
	}
	x.PendingCancel = request
	return request
}

// DraftReport returns a draft [*mkt.Report] for this [*OpenOrder].
func (x *OpenOrder) DraftReport() *mkt.Report {
	return &mkt.Report{
		OrderID:          x.OrderID,
		Symbol:           x.Symbol,
		Side:             x.Side,
		SecondaryOrderID: x.SecondaryOrderID,
		ClOrdID:          x.ClOrdID,
		Account:          x.Account,
		TimeInForce:      x.TimeInForce,
	}
}
