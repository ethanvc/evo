package evolog

import (
	"context"
	"fmt"
	"github.com/ethanvc/evo/base"
	"google.golang.org/grpc/codes"
	"log/slog"
	"runtime"
	"time"
)

func LogRequest(c context.Context, logInfo *RequestLogInfo, extra ...any) {
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
	if logInfo != nil {
		args = append(args, slog.Duration("tc", logInfo.Duration))
	}
	if events := lc.GetEvents(); len(events) > 0 {
		args = append(args, slog.String("events", events))
	}
	Log(c, lvl, 1, "REQ_END", args...)
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

func LogError(c context.Context, event string, args ...any) *base.Status {
	Log(c, slog.LevelError, 1, event, args...)
	return base.New(codes.Internal, event)
}

// Log for increase performance, we use skip as argument.
func Log(c context.Context, lvl slog.Level, skip int, event string, args ...any) {
	l := slog.Default()
	if !l.Enabled(c, lvl) {
		return
	}
	r := slog.NewRecord(time.Now(), lvl, event, GetPC(skip+1))
	r.Add(args...)
	if c == nil {
		c = context.Background()
	}
	_ = l.Handler().Handle(c, r)
}

func GetPC(skip int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(skip+2, pcs[:])
	return pcs[0]
}

func GetCallerLocation(pc uintptr) string {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return fmt.Sprintf("%s:%d", extractTailPart(f.File), f.Line)
}

func extractTailPart(f string) string {
	i := len(f) - 1
	slashCount := 2
	for ; i >= 0; i-- {
		if f[i] == '/' || f[i] == '\\' {
			slashCount--
			if slashCount == 0 {
				return f[i+1:]
			}
		}
	}
	return f
}
