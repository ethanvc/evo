package plog

import (
	"log/slog"
)

func init() {
	sTraceIdInternal = newTraceIdInternal()
	globalLogContext = &LogContext{
		traceId: NewTraceId(),
		method:  "Global",
	}
	initDefaultLog()
}

func initDefaultLog() {
	h := NewJsonHandler(nil)
	l := slog.New(h)
	slog.SetDefault(l)
}

var defaultReporter = NewReporter()

func DefaultReporter() *Reporter {
	return defaultReporter
}
