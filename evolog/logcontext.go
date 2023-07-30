package evolog

import (
	"bytes"
	"context"
	"sync"
)

type LogContext struct {
	TraceId string
	Method  string
	mux     sync.Mutex
	events  bytes.Buffer
}

func (lc *LogContext) AddEvent(event string) *LogContext {
	if lc == globalLogContext {
		return lc
	}
	lc.mux.Lock()
	defer lc.mux.Unlock()
	if lc.events.Len() < 200 {
		lc.events.WriteByte(';')
		lc.events.WriteString(event)
	}
	return lc
}

func (lc *LogContext) GetEvents() string {
	lc.mux.Lock()
	defer lc.mux.Unlock()
	return lc.events.String()
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
