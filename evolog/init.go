package evolog

import (
	"sync/atomic"

	"log/slog"
)

func init() {
	initTraceIdSeed()
	globalLogContext = &LogContext{
		traceId: NewTraceId(),
		method:  "Global",
	}
	SetDefaultReporter(NewReporter(&ReporterConfig{
		Svr:      "not_set",
		Instance: "not_set",
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

func SetDefaultReporter(r *Reporter) {
	if r == nil {
		return
	}
	defaultReporter.Store(r)
}
