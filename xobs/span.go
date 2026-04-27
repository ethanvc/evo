package xobs

import (
	"context"
	"time"
)

type Span struct {
	method       string
	startTime    time.Time
	cost         time.Duration
	traceId      string
	spanId       string
	parentSpanId string
}

type SpanConfig struct {
	Method       string
	TraceId      string
	SpanId       string
	ParentSpanId string
	ObsConfig
}

func WithSpanContext(ctx context.Context, config *SpanConfig) context.Context {
	span := &Span{}
	span.init(ctx, config)
	ctx = WithObsContext(ctx, &config.ObsConfig)
	obsCtx := GetObsContext(ctx)
	obsCtx.span = span
	return ctx
}

func NewSpan(ctx context.Context, config *SpanConfig) *Span {
	span := &Span{}
	span.init(ctx, config)
	return span
}

func (s *Span) init(ctx context.Context, config *SpanConfig) {
	s.method = config.Method
	s.startTime = time.Now()
	if config.TraceId != "" {
		s.traceId = config.TraceId
		s.spanId = config.SpanId
		s.parentSpanId = config.ParentSpanId
	} else if parentSpan := GetObsContext(ctx).getSpan(); parentSpan != nil {
		s.traceId = parentSpan.traceId
		s.parentSpanId = parentSpan.spanId
		s.spanId = generateSpanIdFunc(false)
	} else {
		s.traceId = generateTraceIdFunc()
		s.spanId = generateSpanIdFunc(false)
		s.parentSpanId = generateSpanIdFunc(true)
	}
}

func (s *Span) GetMethod() string {
	return s.method
}

func (s *Span) GetStartTime() time.Time {
	return s.startTime
}

func (s *Span) GetTraceId() string {
	return s.traceId
}

func (s *Span) GetSpanId() string {
	return s.spanId
}

func (s *Span) GetParentSpanId() string {
	return s.parentSpanId
}

func (s *Span) SetAttr(key string, val any) {

}
