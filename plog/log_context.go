package plog

import (
	"bytes"
	"context"
	"sync"
	"time"
)

type LogContext struct {
	traceId   string
	method    string
	startTime time.Time
	mux       sync.Mutex
	events    bytes.Buffer
}

func (lc *LogContext) AddEvent(event string) *LogContext {
	if lc == globalLogContext {
		return lc
	}
	lc.mux.Lock()
	defer lc.mux.Unlock()
	l := lc.events.Len()
	if l < 200 {
		if l != 0 {
			lc.events.WriteByte(';')
		}
		lc.events.WriteString(event)
	}
	return lc
}

func (lc *LogContext) GetTraceId() string {
	lc.mux.Lock()
	defer lc.mux.Unlock()
	return lc.traceId
}

func (lc *LogContext) SetMethod(method string) *LogContext {
	if lc == globalLogContext {
		return lc
	}
	lc.mux.Lock()
	defer lc.mux.Unlock()
	lc.method = method
	return lc
}

func (lc *LogContext) GetMethod() string {
	lc.mux.Lock()
	defer lc.mux.Unlock()
	return lc.method
}

func (lc *LogContext) GetEvents() string {
	lc.mux.Lock()
	defer lc.mux.Unlock()
	return lc.events.String()
}

func (lc *LogContext) GetStartTime() time.Time {
	return lc.startTime
}

func WithLogContext(c context.Context, lcc *LogContextConfig) context.Context {
	if c == nil {
		c = context.Background()
	}
	if lcc == nil {
		lcc = &LogContextConfig{}
	}
	lc := &LogContext{traceId: lcc.TraceId, method: lcc.Method}
	if len(lc.traceId) == 0 {
		lc.traceId = NewTraceId()
	}
	lc.startTime = time.Now()
	return context.WithValue(c, contextKeyLogContext{}, lc)
}

func GetLogContext(c context.Context) *LogContext {
	// we never crash app, even user give me nil context
	if c == nil {
		return globalLogContext
	}
	v, _ := c.Value(contextKeyLogContext{}).(*LogContext)
	if v != nil {
		return v
	}
	return globalLogContext
}

var globalLogContext *LogContext

type contextKeyLogContext struct{}

type LogContextConfig struct {
	Method  string
	TraceId string
}
