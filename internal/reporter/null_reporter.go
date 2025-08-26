package reporter

import "context"

// a noop reporter
type nullReporter struct{}

func newNullReporter() *nullReporter                      { return &nullReporter{} }
func (r *nullReporter) ReportEvent(Event) error           { return nil }
func (r *nullReporter) ReportStatus(Event) error          { return nil }
func (r *nullReporter) Shutdown(context.Context) error    { return nil }
func (r *nullReporter) ShutdownNow()                      {}
func (r *nullReporter) Closed() bool                      { return true }
func (r *nullReporter) WaitForReady(context.Context) bool { return true }
func (r *nullReporter) SetServiceKey(string) error        { return nil }
func (r *nullReporter) GetServiceName() string            { return "" }
