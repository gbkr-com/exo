package dma

import (
	"github.com/gbkr-com/mkt"
)

// OnReport applies the execution report to the [*OpenOrder].
func OnReport(open *OpenOrder, report *mkt.Report) {

	if open == nil || report == nil {
		return
	}

	switch report.OrdStatus {

	case mkt.OrdStatusNew:
		if open.PendingNew == nil {
			// TODO notify
			return
		}
		open.PendingNew.Accept(report.SecondaryOrderID)
		return

	case mkt.OrdStatusPartiallyFilled:
		return

	case mkt.OrdStatusFilled:
		open.Complete = true
		return

	case mkt.OrdStatusCanceled:
		if open.PendingCancel == nil {
			// TODO notify
			return
		}
		open.PendingCancel.Accept()
		open.Complete = true
		return

	case mkt.OrdStatusPendingCancel:
		return

	case mkt.OrdStatusRejected:
		switch {
		case open.PendingNew != nil:
			open.PendingNew.Reject()
			open.Complete = true

		case open.PendingReplace != nil:
			open.PendingReplace.Reject()

		case open.PendingCancel != nil:
			open.PendingCancel.Reject()
		}
		return

	case mkt.OrdStatusPendingNew:
		return

	case mkt.OrdStatusExpired:
		open.Complete = true
		return

	case mkt.OrdStatusPendingReplace:
		return

	default:
		// TODO notify
		return
	}

}
