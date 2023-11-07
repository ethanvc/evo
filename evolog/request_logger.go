package evolog

import (
	"github.com/ethanvc/evo/base"
	"google.golang.org/grpc/codes"
	"log/slog"
	"time"
)
import "context"

type RequestLogger struct {
	filter   func(c context.Context, err error, req, resp any) slog.Level
	logger   *slog.Logger
	reporter *Reporter
}

func NewRequestLogger(
	filter func(c context.Context, err error, req, resp any) slog.Level,
	logger *slog.Logger) *RequestLogger {
	rl := &RequestLogger{
		filter: filter,
		logger: logger,
	}
	return rl
}

func (rl *RequestLogger) Log(c context.Context, logInfo *RequestLogInfo, extra ...any) {
	if logInfo == nil {
		logInfo = &RequestLogInfo{}
	}
	s := base.Convert(logInfo.Err)
	DefaultReporter().ReportRequest(c, s.GetCode(), s.GetEvent())
	lvl := rl.callFilter(c, logInfo.Err, logInfo.Req, logInfo.Resp)
	if !rl.Enabled(c, lvl) {
		return
	}

	lc := GetLogContext(c)
	var args []any
	args = append(args, slog.String("method", lc.GetMethod()))

	if logInfo.Err != nil {
		args = append(args, Error(logInfo.Err))
	}
	if logInfo.Req != nil {
		args = append(args, slog.Any("req", logInfo.Req))
	}
	if logInfo.Resp != nil {
		args = append(args, slog.Any("resp", logInfo.Resp))
	}

	args = append(args, extra...)
	args = append(args, slog.Int64("tc", time.Now().Sub(lc.GetStartTime()).Microseconds()))
	if events := lc.GetEvents(); len(events) > 0 {
		args = append(args, slog.String("events", events))
	}
	Log(c, lvl, 1, "REQ_END", args...)
}

func (rl *RequestLogger) callFilter(c context.Context, err error, req, resp any) slog.Level {
	if rl.filter != nil {
		return rl.filter(c, err, req, resp)
	} else {
		return slog.LevelInfo
	}
}

func (rl *RequestLogger) Enabled(c context.Context, lvl slog.Level) bool {
	if rl.logger != nil {
		return rl.logger.Enabled(c, lvl)
	}
	return slog.Default().Enabled(c, lvl)
}

func (rl *RequestLogger) getReporter() *Reporter {
	if rl.reporter != nil {
		return rl.reporter
	} else {
		return DefaultReporter()
	}
}

type RequestLogInfo struct {
	Err  error
	Req  any
	Resp any
}

var defaultRequestLogger = NewRequestLogger(DefaultFilter, nil)

func DefaultRequestLogger() *RequestLogger {
	return defaultRequestLogger
}

func SetDefaultRequestLogger(l *RequestLogger) {
	if l == nil {
		return
	}
	defaultRequestLogger = l
}

func DefaultFilter(c context.Context, err error, req, resp any) slog.Level {
	s := base.Convert(err)
	if base.In(s.GetCode(),
		codes.Unimplemented, codes.Unknown, codes.Unavailable, codes.Internal, codes.DataLoss,
	) {
		return slog.LevelError
	} else {
		return slog.LevelInfo
	}
}
