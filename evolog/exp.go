package evolog

import (
	"context"
	"github.com/ethanvc/evo/base"
	"log/slog"
	"time"
)

func LogRequest(c context.Context, logInfo *RequestLogInfo, extra ...any) {
	lc := GetLogContext(c)
	lvl := slog.LevelInfo
	var args []any
	args = append(args, slog.String("method", lc.GetMethod()))
	if logInfo != nil {
		if logInfo.Err != nil {
			args = append(args, slog.Any("err", logInfo.Err))
		}
		if logInfo.Req != nil {
			args = append(args, slog.Any("req", logInfo.Req))
		}
		if logInfo.Resp != nil {
			args = append(args, slog.Any("resp", logInfo.Resp))
		}
	}
	args = append(args, extra...)
	if logInfo != nil {
		args = append(args, slog.Duration("tc", logInfo.Duration))
	}
	if events := lc.GetEvents(); len(events) > 0 {
		args = append(args, slog.String("events", events))
	}
	slog.Log(c, lvl, "REQ_END", args...)
}

type RequestLogInfo struct {
	Duration time.Duration
	Err      error
	Req      any
	Resp     any
}

func Error(err error) slog.Attr {
	return NamedError("err", err)
}

func NamedError(k string, err error) slog.Attr {
	if err == nil {
		return slog.Any("err", nil)
	}
	switch realErr := err.(type) {
	case base.StatusError:
		return slog.Any(k, realErr.Status())
	default:
		return slog.String(k, err.Error())
	}
}
