package main

import (
	"sync"
	"time"

	"github.com/gbkr-com/mkt"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
)

// Application implements [quickfix.Application].
type Application struct {
	memoByClOrdID map[string]*Memo
	memoByOrderID map[string]*Memo
	lock          sync.Mutex
}

// NewApplication returns an [*Application] ready to use.
func NewApplication() *Application {
	return &Application{
		memoByClOrdID: map[string]*Memo{},
		memoByOrderID: map[string]*Memo{},
	}
}

// OnCreate notification of a session begin created.
func (x *Application) OnCreate(quickfix.SessionID) {}

// OnLogon notification of a session successfully logging on.
func (x *Application) OnLogon(quickfix.SessionID) {}

// OnLogout notification of a session logging off or disconnecting.
func (x *Application) OnLogout(quickfix.SessionID) {}

// ToAdmin notification of admin message being sent to target.
func (x *Application) ToAdmin(*quickfix.Message, quickfix.SessionID) {}

// ToApp notification of app message being sent to target.
func (x *Application) ToApp(*quickfix.Message, quickfix.SessionID) error {
	return nil
}

// FromAdmin notification of admin message being received from target.
func (x *Application) FromAdmin(*quickfix.Message, quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

// FromApp notification of app message being received from target.
func (x *Application) FromApp(message *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {

	x.lock.Lock()
	defer x.lock.Unlock()

	var msgType field.MsgTypeField
	if reject := message.Header.Get(&msgType); reject != nil {
		return reject
	}

	switch msgType.Value() {
	case enum.MsgType_ORDER_SINGLE:
		return x.handleNewOrder(message, sessionID)
	case enum.MsgType_ORDER_CANCEL_REPLACE_REQUEST:
		return x.handleReplace(message)
	case enum.MsgType_ORDER_CANCEL_REQUEST:
		return x.handleCancel(message, sessionID)
	}

	return quickfix.UnsupportedMessageType()
}

func (x *Application) handleNewOrder(message *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {

	var (
		clOrdID     field.ClOrdIDField
		ordType     field.OrdTypeField
		symbol      field.SymbolField
		side        field.SideField
		orderQty    field.OrderQtyField
		price       field.PriceField
		timeInForce field.TimeInForceField
	)

	if reject := message.Body.Get(&clOrdID); reject != nil {
		return reject
	}

	if reject := message.Body.Get(&ordType); reject != nil {
		return reject
	}
	switch ordType.Value() {
	case enum.OrdType_LIMIT:
	default:
		return quickfix.ValueIsIncorrect(quickfix.Tag(40))
	}

	if reject := message.Body.Get(&symbol); reject != nil {
		return reject
	}

	if reject := message.Body.Get(&side); reject != nil {
		return reject
	}
	switch side.Value() {
	case enum.Side_BUY:
	case enum.Side_SELL:
	default:
		quickfix.ValueIsIncorrect(quickfix.Tag(54))
	}

	if reject := message.Body.Get(&orderQty); reject != nil {
		return reject
	}

	if reject := message.Body.Get(&price); reject != nil {
		return reject
	}

	if reject := message.Body.Get(&timeInForce); reject != nil {
		return reject
	}
	switch timeInForce.Value() {
	case enum.TimeInForce_GOOD_TILL_CANCEL:
	case enum.TimeInForce_IMMEDIATE_OR_CANCEL:
	default:
		return quickfix.ValueIsIncorrect(quickfix.Tag(59))
	}

	memo := &Memo{
		OrderID:     mkt.NewOrderID(),
		ClOrdID:     clOrdID.Value(),
		Symbol:      symbol.Value(),
		Side:        side.Value(),
		OrderQty:    orderQty.Decimal,
		Price:       price.Decimal,
		TimeInForce: timeInForce.Value(),
	}

	//
	// Pending new.
	//
	reply := quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewOrderID(memo.OrderID))
	reply.Body.Set(field.NewClOrdID(clOrdID.Value()))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewSymbol(memo.Symbol))
	reply.Body.Set(field.NewSide(memo.Side))
	reply.Body.Set(field.NewLeavesQty(memo.OrderQty, mkt.Precision(memo.OrderQty))) //
	reply.Body.Set(field.NewCumQty(decimal.Zero, 0))                                //
	reply.Body.Set(field.NewAvgPx(decimal.Zero, 0))                                 //
	reply.Body.Set(field.NewExecType(enum.ExecType_PENDING_NEW))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_PENDING_NEW))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	//
	if err := quickfix.SendToTarget(reply, sessionID); err != nil {
		return nil // TODO
	}
	//
	// New.
	//
	reply = quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewOrderID(memo.OrderID))
	reply.Body.Set(field.NewClOrdID(clOrdID.Value()))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewSymbol(memo.Symbol))
	reply.Body.Set(field.NewSide(memo.Side))
	reply.Body.Set(field.NewLastQty(memo.OrderQty, mkt.Precision(memo.OrderQty)))   //
	reply.Body.Set(field.NewLastPx(memo.Price, mkt.Precision(memo.Price)))          //
	reply.Body.Set(field.NewLeavesQty(memo.OrderQty, mkt.Precision(memo.OrderQty))) //
	reply.Body.Set(field.NewCumQty(decimal.Zero, 0))                                //
	reply.Body.Set(field.NewAvgPx(decimal.Zero, 0))                                 //
	reply.Body.Set(field.NewExecType(enum.ExecType_NEW))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_NEW))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	//
	if err := quickfix.SendToTarget(reply, sessionID); err != nil {
		return nil // TODO
	}

	if timeInForce.Value() == enum.TimeInForce_GOOD_TILL_CANCEL {
		x.memoByClOrdID[memo.ClOrdID] = memo
		x.memoByOrderID[memo.OrderID] = memo
		return nil
	}

	reply = quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewOrderID(memo.OrderID))
	reply.Body.Set(field.NewClOrdID(clOrdID.Value()))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewSymbol(memo.Symbol))
	reply.Body.Set(field.NewSide(memo.Side))
	reply.Body.Set(field.NewLeavesQty(decimal.Zero, 0))                          //
	reply.Body.Set(field.NewCumQty(memo.OrderQty, mkt.Precision(memo.OrderQty))) //
	reply.Body.Set(field.NewAvgPx(memo.Price, mkt.Precision(memo.Price)))        //
	reply.Body.Set(field.NewExecType(enum.ExecType_TRADE))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_FILLED))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	//
	if err := quickfix.SendToTarget(reply, sessionID); err != nil {
		return nil // TODO
	}

	return nil
}

func (x *Application) handleReplace(*quickfix.Message) quickfix.MessageRejectError {
	return nil
}

func (x *Application) handleCancel(message *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {

	var (
		origClOrdID field.OrigClOrdIDField
		clOrdID     field.ClOrdIDField
		symbol      field.SymbolField
		side        field.SideField
		orderQty    field.OrderQtyField
	)

	if reject := message.Body.Get(&origClOrdID); reject != nil {
		return reject
	}
	if reject := message.Body.Get(&clOrdID); reject != nil {
		return reject
	}
	if reject := message.Body.Get(&symbol); reject != nil {
		return reject
	}
	if reject := message.Body.Get(&side); reject != nil {
		return reject
	}
	if reject := message.Body.Get(&orderQty); reject != nil {
		return reject
	}

	memo := x.memoByClOrdID[origClOrdID.Value()]
	if memo == nil {
		reply := quickfix.NewMessage()
		reply.Header.Set(field.NewMsgType(enum.MsgType_ORDER_CANCEL_REJECT))
		reply.Body.Set(field.NewOrderID("NONE"))
		reply.Body.Set(field.NewClOrdID(clOrdID.Value()))
		reply.Body.Set(field.NewOrigClOrdID(origClOrdID.Value()))
		reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_REJECTED))
		reply.Body.Set(field.NewCxlRejResponseTo(enum.CxlRejResponseTo_ORDER_CANCEL_REQUEST))
		reply.Body.Set(field.NewCxlRejReason(enum.CxlRejReason_UNKNOWN_ORDER))
		reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
		if err := quickfix.SendToTarget(reply, sessionID); err != nil {
			return nil // TODO
		}
		return nil
	}

	if memo.Symbol != symbol.Value() || memo.Side != side.Value() || !memo.OrderQty.Equal(orderQty.Decimal) {
		reply := quickfix.NewMessage()
		reply.Header.Set(field.NewMsgType(enum.MsgType_ORDER_CANCEL_REJECT))
		reply.Body.Set(field.NewOrderID(memo.OrderID))
		reply.Body.Set(field.NewClOrdID(clOrdID.Value()))
		reply.Body.Set(field.NewOrigClOrdID(origClOrdID.Value()))
		reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_REJECTED))
		reply.Body.Set(field.NewCxlRejResponseTo(enum.CxlRejResponseTo_ORDER_CANCEL_REQUEST))
		reply.Body.Set(field.NewCxlRejReason(enum.CxlRejReason_OTHER))
		reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
		if err := quickfix.SendToTarget(reply, sessionID); err != nil {
			return nil // TODO
		}
		return nil
	}

	reply := quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewOrderID(memo.OrderID))
	reply.Body.Set(field.NewClOrdID(clOrdID.Value()))
	reply.Body.Set(field.NewOrigClOrdID(origClOrdID.Value()))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewSymbol(memo.Symbol))
	reply.Body.Set(field.NewSide(memo.Side))
	reply.Body.Set(field.NewLeavesQty(memo.OrderQty, mkt.Precision(memo.OrderQty))) //
	reply.Body.Set(field.NewCumQty(decimal.Zero, 0))                                //
	reply.Body.Set(field.NewAvgPx(decimal.Zero, 0))                                 //
	reply.Body.Set(field.NewExecType(enum.ExecType_PENDING_CANCEL))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_PENDING_CANCEL))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	//
	if err := quickfix.SendToTarget(reply, sessionID); err != nil {
		return nil // TODO
	}

	reply = quickfix.NewMessage()
	reply.Header.Set(field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	reply.Body.Set(field.NewOrderID(memo.OrderID))
	reply.Body.Set(field.NewClOrdID(clOrdID.Value()))
	reply.Body.Set(field.NewOrigClOrdID(origClOrdID.Value()))
	reply.Body.Set(field.NewExecID(mkt.NewOrderID()))
	reply.Body.Set(field.NewSymbol(memo.Symbol))
	reply.Body.Set(field.NewSide(memo.Side))
	reply.Body.Set(field.NewLeavesQty(memo.OrderQty, mkt.Precision(memo.OrderQty))) //
	reply.Body.Set(field.NewCumQty(decimal.Zero, 0))                                //
	reply.Body.Set(field.NewAvgPx(decimal.Zero, 0))                                 //
	reply.Body.Set(field.NewExecType(enum.ExecType_CANCELED))
	reply.Body.Set(field.NewOrdStatus(enum.OrdStatus_CANCELED))
	reply.Body.Set(field.NewTransactTime(time.Now().UTC()))
	//
	if err := quickfix.SendToTarget(reply, sessionID); err != nil {
		return nil // TODO
	}

	delete(x.memoByClOrdID, memo.ClOrdID)
	delete(x.memoByOrderID, memo.OrderID)

	return nil
}
