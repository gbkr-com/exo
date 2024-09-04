package fix

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/mkt"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/quickfix"
)

var (
	rejectUnknownClOrdID     = quickfix.NewBusinessMessageRejectError("ClOrdID not known", 380, nil)
	rejectUnknownOrigClOrdID = quickfix.NewBusinessMessageRejectError("OrigClOrdID not known", 380, nil)
	rejectNotPendingNew      = quickfix.NewBusinessMessageRejectError("Order not PENDING_NEW", 380, nil)
	rejectNotPendingReplace  = quickfix.NewBusinessMessageRejectError("Order not PENDING_REPLACE", 380, nil)
	rejectNotPendingCancel   = quickfix.NewBusinessMessageRejectError("Order not PENDING_CANCEL", 380, nil)
	rejectExpireGTC          = quickfix.NewBusinessMessageRejectError("Cannot expire GTC", 380, nil)
)

// Application implements [quickfix.Application] for sending requests and
// receiving execution reports for a single FIX connection.
type Application struct {
	sessionID       quickfix.SessionID
	ordersByClOrdID map[string]*dma.OpenOrder
	ordersByOrderID map[string][]*dma.OpenOrder
	onReport        func(*mkt.Report)
	lock            sync.Mutex
}

// NewApplication returns an [*Application] ready to use.
func NewApplication(onReport func(*mkt.Report)) *Application {
	return &Application{
		ordersByClOrdID: map[string]*dma.OpenOrder{},
		ordersByOrderID: map[string][]*dma.OpenOrder{},
		onReport:        onReport,
	}
}

// SendNew sends the [*NewRequest] to the counterparty.
func (x *Application) SendNew(request *dma.NewRequest) error {

	x.lock.Lock()
	defer x.lock.Unlock()

	x.ordersByClOrdID[request.ClOrdID] = request.OpenOrder
	list := x.ordersByOrderID[request.OpenOrder.OrderID]
	x.ordersByOrderID[request.OpenOrder.OrderID] = append(list, request.OpenOrder)

	message := request.AsQuickFIX()
	return quickfix.SendToTarget(message, x.sessionID)

}

// SendReplace sends the [*ReplaceRequest] to the counterparty.
func (x *Application) SendReplace(request *dma.ReplaceRequest) error {

	x.lock.Lock()
	defer x.lock.Unlock()

	if _, ok := x.ordersByClOrdID[request.OrigClOrdID]; !ok {
		return fmt.Errorf("fix.Application: dma.ReplaceRequest: OrigClOrdID %s not found", request.OrigClOrdID)
	}

	message := request.AsQuickFIX()
	return quickfix.SendToTarget(message, x.sessionID)

}

// SendCancel sends the [*CancelRequest] to the counterparty.
func (x *Application) SendCancel(request *dma.CancelRequest) error {

	x.lock.Lock()
	defer x.lock.Unlock()

	if _, ok := x.ordersByClOrdID[request.OrigClOrdID]; !ok {
		return fmt.Errorf("fix.Application: dma.CancelRequest: OrigClOrdID %s not found", request.OrigClOrdID)
	}

	message := request.AsQuickFIX()
	return quickfix.SendToTarget(message, x.sessionID)

}

// OnCreate implements [quickfix.Application].
func (x *Application) OnCreate(sessionID quickfix.SessionID) {
	x.sessionID = sessionID
}

// OnLogon implements [quickfix.Application].
func (x *Application) OnLogon(quickfix.SessionID) {}

// OnLogout implements [quickfix.Application].
func (x *Application) OnLogout(quickfix.SessionID) {}

// ToAdmin implements [quickfix.Application].
func (x *Application) ToAdmin(*quickfix.Message, quickfix.SessionID) {}

// ToApp implements [quickfix.Application].
func (x *Application) ToApp(*quickfix.Message, quickfix.SessionID) error {
	return nil
}

// FromAdmin implements [quickfix.Application].
func (x *Application) FromAdmin(*quickfix.Message, quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

// FromApp implements [quickfix.Application].
func (x *Application) FromApp(message *quickfix.Message, _ quickfix.SessionID) quickfix.MessageRejectError {

	x.lock.Lock()
	defer x.lock.Unlock()

	var msgType field.MsgTypeField
	reject := message.Header.Get(&msgType)
	if reject != nil {
		return reject
	}

	switch msgType.Value() {
	case enum.MsgType_ORDER_CANCEL_REJECT:
		return x.handleOrderCancelReject(message)
	case enum.MsgType_EXECUTION_REPORT:
		return x.handleExecutionReport(message)
	}

	return nil

}

func (x *Application) handleOrderCancelReject(message *quickfix.Message) quickfix.MessageRejectError {

	var (
		clOrdID          field.ClOrdIDField
		origClOrdID      field.OrigClOrdIDField
		cxlRejResponseTo field.CxlRejResponseToField
		transactTime     field.TransactTimeField
	)
	if reject := message.Body.Get(&clOrdID); reject != nil {
		return reject
	}
	if reject := message.Body.Get(&origClOrdID); reject != nil {
		return reject
	}
	if reject := message.Body.Get(&cxlRejResponseTo); reject != nil {
		return reject
	}

	if reject := message.Body.Get(&transactTime); reject != nil {
		transactTime = field.NewTransactTime(time.Now().UTC())
	}

	open := x.ordersByClOrdID[origClOrdID.Value()]
	if open == nil {
		return nil // TODO
	}
	switch cxlRejResponseTo.Value() {

	case enum.CxlRejResponseTo_ORDER_CANCEL_REPLACE_REQUEST:
		if open.PendingReplace == nil {
			return nil // TODO
		}
		open.PendingReplace.Reject()

	case enum.CxlRejResponseTo_ORDER_CANCEL_REQUEST:
		if open.PendingCancel == nil {
			return nil // TODO
		}
		open.PendingCancel.Reject()

	}

	report := open.DraftReport()
	report.OrdStatus = x.reportFillOrdStatus(message, open)
	report.TransactTime = transactTime.Time
	report.ExecInst = x.reportExecInst(open.OrderID)
	x.onReport(report)

	return nil
}

func (x *Application) handleExecutionReport(message *quickfix.Message) quickfix.MessageRejectError {

	var (
		clOrdID field.ClOrdIDField
		// origClOrdID  field.OrigClOrdIDField
		orderID      field.OrderIDField
		ordStatus    field.OrdStatusField
		execType     field.ExecTypeField
		transactTime field.TransactTimeField
	)
	if reject := message.Body.Get(&clOrdID); reject != nil {
		return reject
	}
	if reject := message.Body.Get(&ordStatus); reject != nil {
		return reject
	}
	if reject := message.Body.Get(&execType); reject != nil {
		return reject
	}
	//
	// Fields which can be defaulted.
	//
	// if reject := message.Body.Get(&origClOrdID); reject != nil {
	// 	origClOrdID = field.NewOrigClOrdID("")
	// }
	if reject := message.Body.Get(&transactTime); reject != nil {
		transactTime = field.NewTransactTime(time.Now().UTC())
	}

	switch execType.Value() {

	case enum.ExecType_PENDING_NEW: // -----------------------------------------
		//
		// A pending new request has been received by the counterparty. Not all
		// counterparties will return this ExecType value but will instead
		// skip directly to ExecType_NEW or ExecType_REJECTED.
		//
		open := x.ordersByClOrdID[clOrdID.Value()]
		if open == nil {
			return rejectUnknownClOrdID
		}
		if reject := message.Body.Get(&orderID); reject == nil {
			open.SecondaryOrderID = orderID.Value()
		}

		report := open.DraftReport()
		report.OrdStatus = mkt.OrdStatusPendingNew
		report.TransactTime = transactTime.Time
		//
		// No need to assign ExecInst as this is itself a pending request.
		//
		x.onReport(report)

	case enum.ExecType_NEW: // -------------------------------------------------
		//
		// A pending new request has been accepted by the counterparty.
		//
		open := x.ordersByClOrdID[clOrdID.Value()]
		if open == nil {
			return rejectUnknownClOrdID
		}
		if reject := message.Body.Get(&orderID); reject == nil {
			open.SecondaryOrderID = orderID.Value()
		}
		if open.PendingNew == nil {
			return rejectNotPendingNew
		}

		open.PendingNew.Accept(orderID.Value())

		report := open.DraftReport()
		report.OrdStatus = mkt.OrdStatusNew
		report.TransactTime = transactTime.Time
		report.ExecInst = x.reportExecInst(open.OrderID)
		x.onReport(report)

	case enum.ExecType_REJECTED: // --------------------------------------------
		//
		// A pending new request, even a new order, has been rejected by the
		// counterparty.
		//
		open := x.ordersByClOrdID[clOrdID.Value()]
		if open == nil {
			return rejectUnknownClOrdID
		}
		if open.PendingNew != nil {
			open.PendingNew.Reject()

		}
		x.remove(clOrdID.Value(), open.OrderID)

		report := open.DraftReport()
		report.ClOrdID = clOrdID.Value()
		report.OrdStatus = mkt.OrdStatusRejected
		report.TransactTime = transactTime.Time
		report.ExecInst = x.reportExecInst(open.OrderID)
		x.onReport(report)

	case enum.ExecType_PENDING_CANCEL: // --------------------------------------
		//
		// A cancel request has been received by the counterparty.
		//
		var origClOrdID field.OrigClOrdIDField
		if reject := message.Body.Get(&origClOrdID); reject != nil {
			return reject
		}
		open := x.ordersByClOrdID[origClOrdID.Value()]
		if open == nil {
			return rejectUnknownOrigClOrdID
		}
		if open.PendingCancel == nil {
			return rejectNotPendingCancel
		}

		report := open.DraftReport()
		report.OrdStatus = mkt.OrdStatusPendingCancel
		report.TransactTime = transactTime.Time
		//
		// No need to assign ExecInst as this is itself a pending request.
		//
		x.onReport(report)

	case enum.ExecType_CANCELED: // --------------------------------------------
		//
		// A cancel request has been accepted by the counterparty.
		//
		var origClOrdID field.OrigClOrdIDField
		if reject := message.Body.Get(&origClOrdID); reject != nil {
			return reject
		}
		open := x.ordersByClOrdID[origClOrdID.Value()]
		if open == nil {
			return rejectUnknownOrigClOrdID
		}
		if open.PendingCancel == nil {
			return rejectNotPendingCancel
		}

		open.PendingCancel.Accept()
		x.remove(origClOrdID.Value(), open.OrderID)

		report := open.DraftReport()
		report.OrdStatus = mkt.OrdStatusCanceled
		report.TransactTime = transactTime.Time
		report.ExecInst = x.reportExecInst(open.OrderID)
		x.onReport(report)

	case enum.ExecType_EXPIRED: // ---------------------------------------------
		//
		// An IOC order has expired.
		//
		open := x.ordersByClOrdID[clOrdID.Value()]
		if open == nil {
			return rejectUnknownClOrdID
		}
		if open.TimeInForce != mkt.IOC {
			return rejectExpireGTC
		}

		x.remove(open.ClOrdID, open.OrderID)

		report := open.DraftReport()
		report.OrdStatus = mkt.OrdStatusExpired
		report.TransactTime = transactTime.Time
		report.ExecInst = x.reportExecInst(open.OrderID)
		x.onReport(report)

	case enum.ExecType_PENDING_REPLACE: // -------------------------------------
		//
		// A replace request has been received by the counterparty.
		//
		open := x.ordersByClOrdID[clOrdID.Value()]
		if open == nil {
			return rejectUnknownClOrdID
		}
		if open.PendingReplace == nil {
			return rejectNotPendingReplace
		}

		report := open.DraftReport()
		report.OrdStatus = mkt.OrdStatusPendingReplace
		report.TransactTime = transactTime.Time
		//
		// No need to assign ExecInst as this is itself a pending request.
		//
		x.onReport(report)

	case enum.ExecType_REPLACED: // --------------------------------------------
		//
		// A replace request has been accepted by the counterparty.
		//
		open := x.ordersByClOrdID[clOrdID.Value()]
		if open == nil {
			return rejectUnknownClOrdID
		}
		if open.PendingReplace == nil {
			return rejectNotPendingReplace
		}

		original := open.ClOrdID
		if reject := message.Body.Get(&orderID); reject == nil {
			open.PendingReplace.Accept(orderID.Value())
		} else {
			open.PendingReplace.Accept("")
		}
		//
		// Accepting promotes the request ClOrdID to the open order ClOrdID.
		//
		delete(x.ordersByClOrdID, original)
		x.ordersByClOrdID[open.ClOrdID] = open

		report := open.DraftReport()
		report.OrdStatus = x.reportFillOrdStatus(message, open)
		report.TransactTime = transactTime.Time
		report.ExecInst = x.reportExecInst(open.OrderID)

	case enum.ExecType_TRADE: // -----------------------------------------------
		//
		// An open order has a fill.
		//
		open := x.ordersByClOrdID[clOrdID.Value()]
		if open == nil {
			return rejectUnknownClOrdID
		}

		var (
			lastQty field.LastQtyField
			lastPx  field.LastPxField
		)
		if reject := message.Body.Get(&lastQty); reject != nil {
			return reject
		}
		if reject := message.Body.Get(&lastPx); reject != nil {
			return reject
		}

		report := open.DraftReport()
		report.OrdStatus = x.reportFillOrdStatus(message, open)
		report.LastQty = lastQty.Decimal
		report.LastPx = lastPx.Decimal
		report.TransactTime = transactTime.Time
		report.ExecInst = x.reportExecInst(open.OrderID)
		x.onReport(report)

		if report.OrdStatus == mkt.OrdStatusFilled {
			x.remove(clOrdID.Value(), open.OrderID)
		}

		//
		// Unsupported values.
		//
	case enum.ExecType_CALCULATED,
		enum.ExecType_DONE_FOR_DAY,
		enum.ExecType_FILL,
		enum.ExecType_ORDER_STATUS,
		enum.ExecType_PARTIAL_FILL,
		enum.ExecType_RESTATED,
		enum.ExecType_STOPPED,
		enum.ExecType_SUSPENDED,
		enum.ExecType_TRADE_CANCEL,
		enum.ExecType_TRADE_CORRECT,
		enum.ExecType_TRADE_HAS_BEEN_RELEASED_TO_CLEARING,
		enum.ExecType_TRADE_IN_A_CLEARING_HOLD,
		enum.ExecType_TRIGGERED_OR_ACTIVATED_BY_SYSTEM:
	}

	return nil

}

func (x *Application) reportFillOrdStatus(message *quickfix.Message, open *dma.OpenOrder) mkt.OrdStatus {
	//
	// For use after a pending replace or pending cancel reject.
	//
	var leavesQty field.LeavesQtyField
	if reject := message.Body.Get(&leavesQty); reject != nil {
		return 0 // TODO
	}
	if leavesQty.Decimal.IsZero() {
		return mkt.OrdStatusFilled
	}
	if leavesQty.Decimal.LessThan(open.OrderQty) {
		return mkt.OrdStatusPartiallyFilled
	}
	return mkt.OrdStatusNew
}

func (x *Application) reportExecInst(orderID string) string {

	for _, open := range x.ordersByOrderID[orderID] {
		if open.IsPending() {
			return ""
		}
		if open.TimeInForce == mkt.IOC {
			return ""
		}
	}
	return "e"

}

func (x *Application) remove(clOrdID, orderID string) {

	delete(x.ordersByClOrdID, clOrdID)

	list := x.ordersByOrderID[orderID]
	list = slices.DeleteFunc(list, func(o *dma.OpenOrder) bool { return o.ClOrdID == clOrdID })
	if len(list) == 0 {
		delete(x.ordersByOrderID, orderID)
		return
	}
	x.ordersByOrderID[orderID] = list

}
