package dma

import (
	"github.com/gbkr-com/mkt"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
)

// A OpenOrder is the current state of an order sent to a counterparty.
type OpenOrder struct {
	Account        string          // FIX field 1
	OrderID        string          // FIX field 37
	ClOrdID        string          // FIX field 11
	Side           mkt.Side        // FIX field 54
	Symbol         string          // FIX field 55
	OrderQty       decimal.Decimal // FIX field 38
	Price          decimal.Decimal // FIX field 44
	TimeInForce    mkt.TimeInForce // FIX field 59
	PendingNew     *NewRequest
	PendingReplace *ReplaceRequest
	PendingCancel  *CancelRequest
	Complete       bool
}

// IsPending returns true if there is an outstanding request.
func (x *OpenOrder) IsPending() bool {
	return x.PendingNew != nil || x.PendingReplace != nil || x.PendingCancel != nil
}

// MakeNewRequest returns a [*NewRequest] if the state allows
func (x *OpenOrder) MakeNewRequest() *NewRequest {
	if x.Complete {
		return nil
	}
	if x.IsPending() {
		return nil
	}
	if x.OrderID != "" {
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
	if x.OrderID == "" {
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
	if x.OrderID == "" {
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

// -----------------------------------------------------------------------------

// NewRequest corresponds to a FIX NewOrderSingle.
type NewRequest struct {
	OpenOrder   *OpenOrder
	ClOrdID     string          // FIX field 11
	Side        mkt.Side        // FIX field 54
	Symbol      string          // FIX field 55
	OrderQty    decimal.Decimal // FIX field 38
	Price       decimal.Decimal // FIX field 44
	TimeInForce mkt.TimeInForce // FIX field 59
}

// Accept the request.
func (x *NewRequest) Accept(orderID string) {
	x.OpenOrder.OrderID = orderID
	x.OpenOrder.ClOrdID = x.ClOrdID
	x.OpenOrder.PendingNew = nil
}

// Reject the requuest.
func (x *NewRequest) Reject() {
	x.OpenOrder.PendingNew = nil
}

// AsQuickFIX returns this request as a non-counterparty specific FIX message.
func (x *NewRequest) AsQuickFIX() *quickfix.Message {
	message := quickfix.NewMessage()
	message.Header.Set(field.NewMsgType(enum.MsgType_ORDER_SINGLE))
	message.Body.Set(field.NewClOrdID(x.ClOrdID))
	message.Body.Set(field.NewOrdType(enum.OrdType_LIMIT))
	message.Body.Set(field.NewSymbol(x.Symbol))
	message.Body.Set(x.Side.AsQuickFIX())
	message.Body.Set(field.NewOrderQty(x.OrderQty, mkt.Precision(x.OrderQty)))
	message.Body.Set(field.NewPrice(x.Price, mkt.Precision(x.Price)))
	message.Body.Set(x.TimeInForce.AsQuickFIX())
	return message
}

// -----------------------------------------------------------------------------

// ReplaceRequest corresponds to a FIX OrderCancelReplaceRequest.
type ReplaceRequest struct {
	OpenOrder   *OpenOrder
	ClOrdID     string           // FIX field 11
	OrigClOrdID string           // FIX field 42
	OrderQty    *decimal.Decimal // FIX field 38
	Price       *decimal.Decimal // FIX field 44
}

// Accept the request, possibly with a new OrderID.
func (x *ReplaceRequest) Accept(orderID string) {
	x.OpenOrder.ClOrdID = x.ClOrdID
	if orderID != "" {
		x.OpenOrder.OrderID = orderID
	}
	if x.OrderQty != nil {
		x.OpenOrder.OrderQty = *x.OrderQty
	}
	if x.Price != nil {
		x.OpenOrder.Price = *x.Price
	}
	x.OpenOrder.PendingReplace = nil
}

// Reject the requuest.
func (x *ReplaceRequest) Reject() {
	x.OpenOrder.PendingReplace = nil
}

// AsQuickFIX returns this request as a non-counterparty specific FIX message.
func (x *ReplaceRequest) AsQuickFIX() *quickfix.Message {
	message := quickfix.NewMessage()
	message.Header.Set(field.NewMsgType(enum.MsgType_ORDER_CANCEL_REPLACE_REQUEST))
	message.Body.Set(field.NewClOrdID(x.ClOrdID))
	message.Body.Set(field.NewOrigClOrdID(x.OrigClOrdID))
	message.Body.Set(field.NewOrderID(x.OpenOrder.OrderID))
	message.Body.Set(field.NewOrdType(enum.OrdType_LIMIT))
	message.Body.Set(field.NewSymbol(x.OpenOrder.Symbol))
	if x.OrderQty != nil {
		message.Body.Set(field.NewOrderQty(*x.OrderQty, mkt.Precision(*x.OrderQty)))
	}
	if x.Price != nil {
		message.Body.Set(field.NewPrice(*x.Price, mkt.Precision(*x.Price)))
	}
	return message
}

// -----------------------------------------------------------------------------

// CancelRequest corresponds to a FIX OrderCancelRequest.
type CancelRequest struct {
	OpenOrder   *OpenOrder
	ClOrdID     string // FIX field 11
	OrigClOrdID string // FIX field 42
}

// Accept the request.
func (x *CancelRequest) Accept() {
	x.OpenOrder.ClOrdID = x.ClOrdID
	x.OpenOrder.PendingCancel = nil
}

// Reject the requuest.
func (x *CancelRequest) Reject() {
	x.OpenOrder.PendingCancel = nil
}

// AsQuickFIX returns this request as a non-counterparty specific FIX message.
func (x *CancelRequest) AsQuickFIX() *quickfix.Message {
	message := quickfix.NewMessage()
	message.Header.Set(field.NewMsgType(enum.MsgType_ORDER_CANCEL_REQUEST))
	message.Body.Set(field.NewClOrdID(x.ClOrdID))
	message.Body.Set(field.NewOrigClOrdID(x.OrigClOrdID))
	message.Body.Set(field.NewOrderID(x.OpenOrder.OrderID))
	message.Body.Set(field.NewSymbol(x.OpenOrder.Symbol))
	return message
}
