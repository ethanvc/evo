package xobs

import (
	"context"
	"time"
)

type GetLogLvlAndEventFuncT func(err error) (Level, string)

type Reporter interface {
	Report(ctx context.Context, lvl Level, event string, labels ...KV)
	ReportDuration(ctx context.Context, duration time.Duration, labels ...KV)
}

type ObsContext struct {
	parent                  *ObsContext
	span                    *Span
	handler                 Handler
	reporter                Reporter
	lvl                     Level
	getLogLevelAndEventFunc GetLogLvlAndEventFuncT
}

type ctxKeyObsContext struct{}

type ObsConfig struct {
	Handler             Handler
	GetLogLevelAndEvent GetLogLvlAndEventFuncT
	Level               Level
}

func WithObsContext(ctx context.Context, config *ObsConfig) context.Context {
	obsCtx := &ObsContext{
		getLogLevelAndEventFunc: config.GetLogLevelAndEvent,
		lvl:                     config.Level,
		handler:                 config.Handler,
	}
	return context.WithValue(ctx, ctxKeyObsContext{}, obsCtx)
}

func withObsContext(ctx context.Context) (context.Context, *ObsContext) {
	obsCtx := &ObsContext{}
	return context.WithValue(ctx, ctxKeyObsContext{}, obsCtx), obsCtx
}

func GetObsContext(ctx context.Context) *ObsContext {
	val, _ := ctx.Value(ctxKeyObsContext{}).(*ObsContext)
	return val
}

func GetRootSpan(ctx context.Context) *Span {
	obsCtx := GetObsContext(ctx)
	return obsCtx.GetRootSpan()
}

func (oc *ObsContext) GetRootSpan() *Span {
	var span *Span
	for oc != nil {
		if oc.span != nil {
			span = oc.span
		}
		oc = oc.parent
	}
	if span != nil {
		return span
	}
	return defaultSpan
}

func (oc *ObsContext) GetSpan() *Span {
	span := oc.getSpan()
	if span != nil {
		return span
	}
	return defaultSpan
}

func (oc *ObsContext) getSpan() *Span {
	for oc != nil {
		if oc.span != nil {
			return oc.span
		}
		oc = oc.parent
	}
	return nil
}

func (oc *ObsContext) AccessLogReport(ctx context.Context, err error, req, resp any, labels []KV, args ...any) {
	lvl, event := oc.getLoggerLevelAndEventWrapper(err)
	oc.reportAccessLog(ctx, lvl, event, labels...)
	args2 := append([]any{}, "err", err, "req", req, "resp", resp)
	args2 = append(args2, args...)
	oc.Log(ctx, 1, lvl, "REQ_END", args2...)
}

func (oc *ObsContext) getLoggerLevelAndEventWrapper(err error) (Level, string) {
	occ := oc
	for occ != nil {
		if occ.getLogLevelAndEventFunc != nil {
			return occ.getLogLevelAndEventFunc(err)
		}
		occ = occ.parent
	}
	return GetDefaultGetLogLevelAndEvent()(err)
}

func (oc *ObsContext) SetAttr(key string, val any) {}

func (oc *ObsContext) GetHandler() Handler {
	for oc != nil {
		if oc.handler != nil {
			return oc.handler
		}
		oc = oc.parent
	}
	return defaultHandler
}

func (oc *ObsContext) GetLevel() Level {
	for oc != nil {
		if oc.lvl != LevelNotSet {
			return oc.lvl
		}
		oc = oc.parent
	}
	return GetDefaultLogLevel()
}

func (oc *ObsContext) Enabled(lvl Level) bool {
	return lvl >= oc.GetLevel()
}

func (oc *ObsContext) Log(ctx context.Context, skip int, lvl Level, event string, args ...any) {
	if !oc.Enabled(lvl) {
		return
	}
	item := LogItem{
		Msg:      event,
		Time:     sNow(),
		Level:    lvl,
		Position: GetCallerPosition(skip + 1),
		ObsCtx:   oc,
	}
	item.Add(args...)
	oc.GetHandler().Handle(ctx, item)
}

func (oc *ObsContext) reportAccessLog(ctx context.Context, lvl Level, event string, labels ...KV) {
	reporter := oc.getReporter()
	span := oc.GetSpan()
	labels = append(labels, KV{Key: "method", Val: span.GetMethod()})
	labels = append(labels, KV{Key: "lvl", Val: lvl.String()})
	reporter.Report(ctx, lvl, "REQ_END;"+event, labels...)
	reporter.ReportDuration(ctx, time.Since(span.GetStartTime()), labels...)
}

func (oc *ObsContext) getReporter() Reporter {
	for oc != nil {
		if oc.reporter != nil {
			return oc.reporter
		}
		oc = oc.parent
	}
	return defaultReporter
}
