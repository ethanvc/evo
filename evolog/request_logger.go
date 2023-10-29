package evolog

import (
	"log/slog"
	"time"
)
import "context"

type RequestLogger struct {
}

func NewRequestLogger() *RequestLogger {
	rl := &RequestLogger{}
	return rl
}

func (rl *RequestLogger) Log(c context.Context, logInfo *RequestLogInfo, extra ...any) {
	lc := GetLogContext(c)
	lvl := slog.LevelInfo
	var args []any
	args = append(args, slog.String("method", lc.GetMethod()))
	if logInfo != nil {
		if logInfo.Err != nil {
			args = append(args, Error(logInfo.Err))
		}
		if logInfo.Req != nil {
			args = append(args, slog.Any("req", logInfo.Req))
		}
		if logInfo.Resp != nil {
			args = append(args, slog.Any("resp", logInfo.Resp))
		}
	}
	args = append(args, extra...)
	args = append(args, slog.Int64("tc", time.Now().Sub(lc.GetStartTime()).Microseconds()))
	if events := lc.GetEvents(); len(events) > 0 {
		args = append(args, slog.String("events", events))
	}
	Log(c, lvl, 1, "REQ_END", args...)
}

type RequestLogInfo struct {
	Err  error
	Req  any
	Resp any
}
