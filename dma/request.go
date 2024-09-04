package dma

import (
	"github.com/gbkr-com/mkt"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
)

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

// Accept the request using the order ID from the counterparty.
func (x *NewRequest) Accept(orderID string) {
	x.OpenOrder.SecondaryOrderID = orderID
	x.OpenOrder.PendingNew = nil
}

// Reject the request.
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
		x.OpenOrder.SecondaryOrderID = orderID
	}
	if x.OrderQty != nil {
		x.OpenOrder.OrderQty = *x.OrderQty
	}
	if x.Price != nil {
		x.OpenOrder.Price = *x.Price
	}
	x.OpenOrder.PendingReplace = nil
}

// Reject the request.
func (x *ReplaceRequest) Reject() {
	x.OpenOrder.PendingReplace = nil
}

// AsQuickFIX returns this request as a non-counterparty specific FIX message.
func (x *ReplaceRequest) AsQuickFIX() *quickfix.Message {
	message := quickfix.NewMessage()
	message.Header.Set(field.NewMsgType(enum.MsgType_ORDER_CANCEL_REPLACE_REQUEST))
	message.Body.Set(field.NewClOrdID(x.ClOrdID))
	message.Body.Set(field.NewOrigClOrdID(x.OrigClOrdID))
	message.Body.Set(field.NewOrderID(x.OpenOrder.SecondaryOrderID))
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

// Reject the request.
func (x *CancelRequest) Reject() {
	x.OpenOrder.PendingCancel = nil
}

// AsQuickFIX returns this request as a non-counterparty specific FIX message.
func (x *CancelRequest) AsQuickFIX() *quickfix.Message {
	message := quickfix.NewMessage()
	message.Header.Set(field.NewMsgType(enum.MsgType_ORDER_CANCEL_REQUEST))
	message.Body.Set(field.NewClOrdID(x.ClOrdID))
	message.Body.Set(field.NewOrigClOrdID(x.OrigClOrdID))
	message.Body.Set(field.NewOrderID(x.OpenOrder.SecondaryOrderID))
	message.Body.Set(field.NewSymbol(x.OpenOrder.Symbol))
	return message
}
