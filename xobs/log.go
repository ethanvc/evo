package xobs

import (
	"context"
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
