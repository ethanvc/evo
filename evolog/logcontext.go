package evolog

import "context"

type LogContext struct {
	TraceId string
	Method  string
}

func WithLogContext(c context.Context, traceId string) context.Context {
	lc := &LogContext{TraceId: traceId}
	return context.WithValue(c, contextKeyLogContext{}, lc)
}

func GetLogContext(c context.Context) *LogContext {
	v, _ := c.Value(contextKeyLogContext{}).(*LogContext)
	if v != nil {
		return v
	}
	return globalLogContext
}

var globalLogContext *LogContext

type contextKeyLogContext struct{}
