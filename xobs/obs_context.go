package xobs

import "context"

type GetAccessLogLevelFuncT func(err error) Level

type ObsContext struct {
	parent            *ObsContext
	span              *Span
	handler           Handler
	lvl               Level
	getAccessLogLevel GetAccessLogLevelFuncT
}

type ctxKeyObsContext struct{}

type ObsConfig struct {
	Handler     Handler
	GetLogLevel GetAccessLogLevelFuncT
	Level       Level
}

func WithObsContext(ctx context.Context, config *ObsConfig) context.Context {
	obsCtx := &ObsContext{
		getAccessLogLevel: config.GetLogLevel,
		lvl:               config.Level,
		handler:           config.Handler,
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
	oc.Log(ctx, 1, LevelErr, "", args...)
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
