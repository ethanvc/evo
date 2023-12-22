package plog

import (
	"sync/atomic"

	"log/slog"
)

func init() {
	sTraceIdInternal = newTraceIdInternal()
	globalLogContext = &LogContext{
		traceId: NewTraceId(),
		method:  "Global",
	}
	SetDefaultReporter(NewReporter(&ReporterConfig{
		ReportSvr:  "not_set",
		ReportInst: "not_set",
	}))
	initDefaultLog()
}

func initDefaultLog() {
	h := NewJsonHandler(nil)
	l := slog.New(h)
	slog.SetDefault(l)
}

var defaultReporter atomic.Pointer[Reporter]

func DefaultReporter() *Reporter {
	return defaultReporter.Load()
}

func SetDefaultReporter(r *Reporter) *Reporter {
	if r == nil {
		return nil
	}
	return defaultReporter.Swap(r)
}
