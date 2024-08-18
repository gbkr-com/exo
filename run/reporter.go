package run

import (
	"sync"

	"github.com/gbkr-com/mkt"
)

// Reporter feeds the channel into [Dispatcher].
type Reporter struct {
	reports chan *mkt.Report
	block   sync.WaitGroup
}

// OnReport sends the [*mkt.Report] onto the channel. This will block until
// the report has been acknowledged ([Reporter.Acknowledge]), as reports must
// not be lost. Typically the caller will be ready from a checkpointed stream,
// such as Redis.
func (x *Reporter) OnReport(report *mkt.Report) {
	if report == nil {
		return
	}
	x.block.Wait()
	x.reports <- report
	x.block.Add(1)
}

// Acknowledge signals that the last [*mkt.Report] sent on the channel has been
// processed.
func (x *Reporter) Acknowledge() {
	x.block.Done()
}
