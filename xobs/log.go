package xobs

import (
	"context"
	"fmt"
	"runtime"
)

func ReportInfo(ctx context.Context, event string, labels ...KV) {}

func Info(ctx context.Context, event string, args ...any) {
	obsCtx := GetObsContext(ctx)
	obsCtx.Log(ctx, 1, LevelInfo, event, args...)
}

func ErrReport(ctx context.Context, event string, args ...any) {
}

func Err(ctx context.Context, event string, args ...any) {
	obsCtx := GetObsContext(ctx)
	obsCtx.Log(ctx, 1, LevelErr, event, args...)
}

func ReportErr(ctx context.Context, event string, labels ...KV) {

}

func InvariantErrReport(ctx context.Context, event string, args ...any) {}

func PanicReport(ctx context.Context, event string, args ...any) {
	obsCtx := GetObsContext(ctx)
	obsCtx.Log(ctx, 1, LevelErr, event, args...)
	panic(event)
}

type KV struct {
	Key string
	Val string
}

type Handler interface {
	Handle(ctx context.Context, item LogItem)
}

func GetCallerPosition(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "?:0"
	}
	return fmt.Sprintf("%s:%d", GetFilePathTailPart(file, 2), line)
}
