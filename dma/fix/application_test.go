package fix

import (
	"testing"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/mkt"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGTCPendingNewThenNew(t *testing.T) {

	var (
		blankSessionID quickfix.SessionID
		report         *mkt.Report
	)
	app := NewApplication(func(r *mkt.Report) { report = r })

	//
	// The parent order.
	//
	def := &mkt.Order{
		MsgType: 0,
		OrderID: mkt.NewOrderID(),
		Side:    mkt.Buy,
		Symbol:  "X",
	}
	order := dma.NewOpenOrder(def)
	assert.Equal(t, order.OrderID, def.OrderID)
	assert.Equal(t, def.Side, order.Side)
	assert.Equal(t, def.Symbol, order.Symbol)
	order.OrderQty = decimal.New(100, 0)
	order.Price = decimal.New(42, 0)
	order.TimeInForce = mkt.GTC
	//
	// New request.
	//
	nr := order.MakeNewRequest()
	assert.NotNil(t, app.SendNew(nr), "because there is no real FIX session")
	assert.Equal(t, 1, len(app.ordersByClOrdID))
	assert.Equal(t, 1, len(app.ordersByOrderID))
	assert.NotNil(t, app.ordersByClOrdID[nr.ClOrdID].PendingNew)
	//
	// Pending new execution report.
	//
	secondary := mkt.NewOrderID()
	reply := quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewClOrdID(nr.ClOrdID))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewOrderID(secondary))
	reply.Body.Set(field.NewOrderQty(nr.OrderQty, mkt.Precision(nr.OrderQty)))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_PENDING_NEW))
	reply.Body.Set(field.NewOrdType(enum.OrdType_LIMIT))
	reply.Body.Set(field.NewSide(nr.Side.AsQuickFIX().Value()))
	reply.Body.Set(field.NewSymbol(nr.Symbol))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	reply.Body.Set(field.NewExecType(enum.ExecType_PENDING_NEW))
	reply.Body.Set(field.NewPrice(nr.Price, mkt.Precision(nr.Price)))
	reply.Body.Set(field.NewTimeInForce(enum.TimeInForce_GOOD_TILL_CANCEL))

	assert.Nil(t, app.FromApp(reply, blankSessionID))
	assert.NotNil(t, report)

	assert.Equal(t, def.OrderID, report.OrderID)
	assert.Equal(t, def.Symbol, report.Symbol)
	assert.Equal(t, def.Side, report.Side)
	assert.Equal(t, secondary, report.SecondaryOrderID)
	assert.Equal(t, nr.ClOrdID, report.ClOrdID)
	assert.Equal(t, mkt.OrdStatusPendingNew, report.OrdStatus)
	assert.Equal(t, order.TimeInForce, report.TimeInForce)
	assert.Equal(t, "", report.ExecInst)

	assert.NotNil(t, app.ordersByClOrdID[nr.ClOrdID].PendingNew)

	//
	// New execution report.
	//
	reply = quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewClOrdID(nr.ClOrdID))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewOrderID(secondary))
	reply.Body.Set(field.NewOrderQty(nr.OrderQty, mkt.Precision(nr.OrderQty)))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_NEW))
	reply.Body.Set(field.NewOrdType(enum.OrdType_LIMIT))
	reply.Body.Set(field.NewSide(nr.Side.AsQuickFIX().Value()))
	reply.Body.Set(field.NewSymbol(nr.Symbol))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	reply.Body.Set(field.NewExecType(enum.ExecType_NEW))
	reply.Body.Set(field.NewPrice(nr.Price, mkt.Precision(nr.Price)))
	reply.Body.Set(field.NewTimeInForce(enum.TimeInForce_GOOD_TILL_CANCEL))

	assert.Nil(t, app.FromApp(reply, blankSessionID))
	assert.NotNil(t, report)

	assert.Equal(t, def.OrderID, report.OrderID)
	assert.Equal(t, def.Symbol, report.Symbol)
	assert.Equal(t, def.Side, report.Side)
	assert.Equal(t, secondary, report.SecondaryOrderID)
	assert.Equal(t, nr.ClOrdID, report.ClOrdID)
	assert.Equal(t, mkt.OrdStatusNew, report.OrdStatus)
	assert.Equal(t, order.TimeInForce, report.TimeInForce)
	assert.Equal(t, "e", report.ExecInst, "no longer pending")

	assert.Nil(t, app.ordersByClOrdID[nr.ClOrdID].PendingNew)

}

func TestIOCNewThenExpire(t *testing.T) {

	var (
		blankSessionID quickfix.SessionID
		report         *mkt.Report
	)
	app := NewApplication(func(r *mkt.Report) { report = r })

	//
	// The parent order.
	//
	def := &mkt.Order{
		MsgType: 0,
		OrderID: mkt.NewOrderID(),
		Side:    mkt.Buy,
		Symbol:  "X",
	}
	order := dma.NewOpenOrder(def)
	order.OrderQty = decimal.New(100, 0)
	order.Price = decimal.New(42, 0)
	order.TimeInForce = mkt.IOC
	//
	// New request.
	//
	nr := order.MakeNewRequest()
	assert.NotNil(t, app.SendNew(nr), "because there is no real FIX session")
	assert.Equal(t, 1, len(app.ordersByClOrdID))
	assert.Equal(t, 1, len(app.ordersByOrderID))
	assert.NotNil(t, app.ordersByClOrdID[nr.ClOrdID].PendingNew)
	//
	// New execution report.
	//
	secondary := mkt.NewOrderID()
	reply := quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewClOrdID(nr.ClOrdID))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewOrderID(secondary))
	reply.Body.Set(field.NewOrderQty(nr.OrderQty, mkt.Precision(nr.OrderQty)))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_NEW))
	reply.Body.Set(field.NewOrdType(enum.OrdType_LIMIT))
	reply.Body.Set(field.NewSide(nr.Side.AsQuickFIX().Value()))
	reply.Body.Set(field.NewSymbol(nr.Symbol))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	reply.Body.Set(field.NewExecType(enum.ExecType_NEW))
	reply.Body.Set(field.NewPrice(nr.Price, mkt.Precision(nr.Price)))
	reply.Body.Set(field.NewTimeInForce(enum.TimeInForce_IMMEDIATE_OR_CANCEL))

	assert.Nil(t, app.FromApp(reply, blankSessionID))
	assert.NotNil(t, report)

	assert.Equal(t, def.OrderID, report.OrderID)
	assert.Equal(t, def.Symbol, report.Symbol)
	assert.Equal(t, def.Side, report.Side)
	assert.Equal(t, secondary, report.SecondaryOrderID)
	assert.Equal(t, nr.ClOrdID, report.ClOrdID)
	assert.Equal(t, mkt.OrdStatusNew, report.OrdStatus)
	assert.Equal(t, order.TimeInForce, report.TimeInForce)
	assert.Equal(t, "", report.ExecInst, "IOC still open")

	assert.Nil(t, app.ordersByClOrdID[nr.ClOrdID].PendingNew)

	//
	// Expired execution report.
	//
	reply = quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewClOrdID(nr.ClOrdID))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewOrderID(secondary))
	reply.Body.Set(field.NewOrderQty(nr.OrderQty, mkt.Precision(nr.OrderQty)))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_EXPIRED))
	reply.Body.Set(field.NewOrdType(enum.OrdType_LIMIT))
	reply.Body.Set(field.NewSide(nr.Side.AsQuickFIX().Value()))
	reply.Body.Set(field.NewSymbol(nr.Symbol))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	reply.Body.Set(field.NewExecType(enum.ExecType_EXPIRED))
	reply.Body.Set(field.NewPrice(nr.Price, mkt.Precision(nr.Price)))
	reply.Body.Set(field.NewTimeInForce(enum.TimeInForce_IMMEDIATE_OR_CANCEL))

	assert.Nil(t, app.FromApp(reply, blankSessionID))
	assert.NotNil(t, report)

	assert.Equal(t, def.OrderID, report.OrderID)
	assert.Equal(t, def.Symbol, report.Symbol)
	assert.Equal(t, def.Side, report.Side)
	assert.Equal(t, secondary, report.SecondaryOrderID)
	assert.Equal(t, nr.ClOrdID, report.ClOrdID)
	assert.Equal(t, mkt.OrdStatusExpired, report.OrdStatus)
	assert.Equal(t, order.TimeInForce, report.TimeInForce)
	assert.Equal(t, "e", report.ExecInst, "IOC expired")

	assert.Equal(t, 0, len(app.ordersByClOrdID))
	assert.Equal(t, 0, len(app.ordersByOrderID))

}

func TestRejectPendingNew(t *testing.T) {

	var (
		blankSessionID quickfix.SessionID
		report         *mkt.Report
	)
	app := NewApplication(func(r *mkt.Report) { report = r })

	//
	// The parent order.
	//
	def := &mkt.Order{
		MsgType: 0,
		OrderID: mkt.NewOrderID(),
		Side:    mkt.Buy,
		Symbol:  "X",
	}
	order := dma.NewOpenOrder(def)
	order.OrderQty = decimal.New(100, 0)
	order.Price = decimal.New(42, 0)
	order.TimeInForce = mkt.GTC
	//
	// New request.
	//
	nr := order.MakeNewRequest()
	assert.NotNil(t, app.SendNew(nr), "because there is no real FIX session")
	assert.Equal(t, 1, len(app.ordersByClOrdID))
	assert.Equal(t, 1, len(app.ordersByOrderID))
	assert.NotNil(t, app.ordersByClOrdID[nr.ClOrdID].PendingNew)
	//
	// Pending new execution report.
	//
	secondary := mkt.NewOrderID()
	reply := quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewClOrdID(nr.ClOrdID))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewOrderID(secondary))
	reply.Body.Set(field.NewOrderQty(nr.OrderQty, mkt.Precision(nr.OrderQty)))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_PENDING_NEW))
	reply.Body.Set(field.NewOrdType(enum.OrdType_LIMIT))
	reply.Body.Set(field.NewSide(nr.Side.AsQuickFIX().Value()))
	reply.Body.Set(field.NewSymbol(nr.Symbol))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	reply.Body.Set(field.NewExecType(enum.ExecType_PENDING_NEW))
	reply.Body.Set(field.NewPrice(nr.Price, mkt.Precision(nr.Price)))
	reply.Body.Set(field.NewTimeInForce(enum.TimeInForce_GOOD_TILL_CANCEL))

	assert.Nil(t, app.FromApp(reply, blankSessionID))
	assert.NotNil(t, report)

	assert.Equal(t, def.OrderID, report.OrderID)
	assert.Equal(t, def.Symbol, report.Symbol)
	assert.Equal(t, def.Side, report.Side)
	assert.Equal(t, secondary, report.SecondaryOrderID)
	assert.Equal(t, nr.ClOrdID, report.ClOrdID)
	assert.Equal(t, mkt.OrdStatusPendingNew, report.OrdStatus)
	assert.Equal(t, order.TimeInForce, report.TimeInForce)
	assert.Equal(t, "", report.ExecInst)

	//
	// Rejected execution report.
	//
	reply = quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewClOrdID(nr.ClOrdID))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewOrderID(secondary))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_REJECTED))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	reply.Body.Set(field.NewExecType(enum.ExecType_REJECTED))

	assert.Nil(t, app.FromApp(reply, blankSessionID))
	assert.NotNil(t, report)

	assert.Equal(t, def.OrderID, report.OrderID)
	assert.Equal(t, def.Symbol, report.Symbol)
	assert.Equal(t, def.Side, report.Side)
	assert.Equal(t, secondary, report.SecondaryOrderID)
	assert.Equal(t, nr.ClOrdID, report.ClOrdID)
	assert.Equal(t, mkt.OrdStatusRejected, report.OrdStatus)
	assert.Equal(t, order.TimeInForce, report.TimeInForce)
	assert.Equal(t, "e", report.ExecInst)

	assert.Equal(t, 0, len(app.ordersByClOrdID))
	assert.Equal(t, 0, len(app.ordersByOrderID))

}

func TestRejectNew(t *testing.T) {

	var (
		blankSessionID quickfix.SessionID
		report         *mkt.Report
	)
	app := NewApplication(func(r *mkt.Report) { report = r })

	//
	// The parent order.
	//
	def := &mkt.Order{
		MsgType: 0,
		OrderID: mkt.NewOrderID(),
		Side:    mkt.Buy,
		Symbol:  "X",
	}
	order := dma.NewOpenOrder(def)
	order.OrderQty = decimal.New(100, 0)
	order.Price = decimal.New(42, 0)
	order.TimeInForce = mkt.GTC
	//
	// New request.
	//
	nr := order.MakeNewRequest()
	assert.NotNil(t, app.SendNew(nr), "because there is no real FIX session")
	assert.Equal(t, 1, len(app.ordersByClOrdID))
	assert.Equal(t, 1, len(app.ordersByOrderID))
	assert.NotNil(t, app.ordersByClOrdID[nr.ClOrdID].PendingNew)
	//
	// New execution report.
	//
	secondary := mkt.NewOrderID()
	reply := quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewClOrdID(nr.ClOrdID))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewOrderID(secondary))
	reply.Body.Set(field.NewOrderQty(nr.OrderQty, mkt.Precision(nr.OrderQty)))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_NEW))
	reply.Body.Set(field.NewOrdType(enum.OrdType_LIMIT))
	reply.Body.Set(field.NewSide(nr.Side.AsQuickFIX().Value()))
	reply.Body.Set(field.NewSymbol(nr.Symbol))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	reply.Body.Set(field.NewExecType(enum.ExecType_NEW))
	reply.Body.Set(field.NewPrice(nr.Price, mkt.Precision(nr.Price)))
	reply.Body.Set(field.NewTimeInForce(enum.TimeInForce_GOOD_TILL_CANCEL))

	assert.Nil(t, app.FromApp(reply, blankSessionID))
	assert.NotNil(t, report)

	assert.Equal(t, def.OrderID, report.OrderID)
	assert.Equal(t, def.Symbol, report.Symbol)
	assert.Equal(t, def.Side, report.Side)
	assert.Equal(t, secondary, report.SecondaryOrderID)
	assert.Equal(t, nr.ClOrdID, report.ClOrdID)
	assert.Equal(t, mkt.OrdStatusNew, report.OrdStatus)
	assert.Equal(t, order.TimeInForce, report.TimeInForce)
	assert.Equal(t, "e", report.ExecInst, "GTC is not pending")

	//
	// Rejected execution report.
	//
	reply = quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewClOrdID(nr.ClOrdID))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewOrderID(secondary))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_REJECTED))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	reply.Body.Set(field.NewExecType(enum.ExecType_REJECTED))

	assert.Nil(t, app.FromApp(reply, blankSessionID))
	assert.NotNil(t, report)

	assert.Equal(t, def.OrderID, report.OrderID)
	assert.Equal(t, def.Symbol, report.Symbol)
	assert.Equal(t, def.Side, report.Side)
	assert.Equal(t, secondary, report.SecondaryOrderID)
	assert.Equal(t, nr.ClOrdID, report.ClOrdID)
	assert.Equal(t, mkt.OrdStatusRejected, report.OrdStatus)
	assert.Equal(t, order.TimeInForce, report.TimeInForce)
	assert.Equal(t, "e", report.ExecInst)

	assert.Equal(t, 0, len(app.ordersByClOrdID))
	assert.Equal(t, 0, len(app.ordersByOrderID))

}
