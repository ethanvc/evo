package evolog

import "context"

type LogContext struct {
	TraceId string
}

func WithLogContext(c context.Context, traceId string) context.Context {
	lc := &LogContext{TraceId: traceId}
	return context.WithValue(c, contextKeyLogContext{}, lc)
}

type contextKeyLogContext struct{}
