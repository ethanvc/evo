package obs

import (
	"context"
	"errors"
	"time"
)

type GetLogLvlAndEventFuncT func(err error) (Level, string)

type GetLogErrorObjectT func(err error) *Error
type GetLogErrorLevelT func(err error) Level

type ObsContext struct {
	parent                  *ObsContext
	span                    *Span
	handler                 Handler
	reporter                *Reporter
	lvl                     Level
	getLogLevelAndEventFunc GetLogLvlAndEventFuncT
	GetLogErrorObjectFunc   GetLogErrorObjectT
	GetLogErrorLevelFunc    GetLogErrorLevelT
}

type ctxKeyObsContext struct{}

type ObsConfig struct {
	Handler               Handler
	GetLogLevelAndEvent   GetLogLvlAndEventFuncT
	GetLogErrorObjectFunc GetLogErrorObjectT
	GetLogErrorLevel      GetLogErrorLevelT
	Level                 Level
}

func WithObsContext(ctx context.Context, config *ObsConfig) (context.Context, *ObsContext) {
	obsCtx := &ObsContext{
		getLogLevelAndEventFunc: config.GetLogLevelAndEvent,
		lvl:                     config.Level,
		handler:                 config.Handler,
	}
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
	typErr := oc.getLogErrorObject(err)
	if typErr != nil {
		err = typErr
	}
	lvl := oc.getLogErrorLevel(typErr)
	span := oc.GetSpan()
	tc := time.Since(span.GetStartTime())
	oc.reportAccessLog(ctx, tc, lvl, typErr.GetReportEvent(), labels...)
	var args2 []any
	args2 = append(args2, "method", span.GetMethod())
	args2 = append(args2, "tc", tc)
	args2 = append(args2, "err", err, "req", req, "resp", resp)
	args2 = append(args2, args...)
	args2 = append(args2, "attris", span.GetAttrs())
	oc.Log(ctx, 1, lvl, "REQ_END", args2...)
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

func (oc *ObsContext) reportAccessLog(ctx context.Context, tc time.Duration, lvl Level, event string, labels ...KV) {
	reporter := oc.getReporter()
	span := oc.GetSpan()
	labels = append(labels, KV{Key: "method", Val: span.GetMethod()})
	labels = append(labels, KV{Key: "lvl", Val: lvl.String()})
	reporter.Report(ctx, lvl, "REQ_END;"+event, labels...)
	reporter.ReportDuration(ctx, lvl, event, tc, labels...)
}

func (oc *ObsContext) getReporter() *Reporter {
	for oc != nil {
		if oc.reporter != nil {
			return oc.reporter
		}
		oc = oc.parent
	}
	return defaultReporter
}

func (oc *ObsContext) getLogErrorObject(err error) *Error {
	for oc != nil {
		if oc.GetLogErrorObjectFunc != nil {
			return oc.GetLogErrorObjectFunc(err)
		}
		oc = oc.parent
	}
	return GetLogErrorObject(err)
}

func (oc *ObsContext) getLogErrorLevel(err *Error) Level {
	for oc != nil {
		if oc.GetLogErrorLevelFunc != nil {
			return oc.GetLogErrorLevelFunc(err)
		}
		oc = oc.parent
	}
	return GetLogErrorLevel(err)
}

func GetLogErrorObject(err error) *Error {
	if err == nil {
		return nil
	}
	if realErr, ok := errors.AsType[*Error](err); ok {
		return realErr
	}
	if errors.Is(err, context.Canceled) {
		return New(Canceled, "ContextCanceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return New(DeadlineExceeded, "ContextDeadlineExceeded")
	}
	typErr := New(Unknown, err.Error())
	return typErr
}

func GetLogErrorLevel(err *Error) Level {
	switch err.GetCode() {
	case OK, NotFound, AlreadyExists, InvalidArgument, Unauthenticated, FailedPrecondition:
		return LevelInfo
	default:
		return LevelErr
	}
}
