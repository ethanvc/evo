package evolog

import (
	"context"
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
	args = append(args, slog.String("events", lc.GetEvents()))
	slog.Log(c, lvl, "REQ_END", args...)
}

type RequestLogInfo struct {
	Duration time.Duration
	Err      error
	Req      any
	Resp     any
}
