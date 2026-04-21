package xobs

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
)

func Info(ctx context.Context, event string, args ...any) {
	obsCtx := GetObsContext(ctx)
	obsCtx.LogRaw(ctx, obsCtx, 1, LevelInfo, event, args...)
}

func ErrReport(ctx context.Context, event string, args ...any) {
}

func Err(ctx context.Context, event string, args ...any) {
	obsCtx := GetObsContext(ctx)
	obsCtx.LogRaw(ctx, obsCtx, 1, LevelErr, event, args...)
}

func ReportErr(ctx context.Context, event string, labels ...KV) {

}

func PanicReport(ctx context.Context, event string, args ...any) {}

type KV struct {
	Key string
	Val string
}

type Handler interface {
	Handle(ctx context.Context, item LogItem)
}

// lastTwoPathParts 返回路径的最后两段（如 pkg/foo.go）；不足两段则只返回最后一段。
func lastTwoPathParts(file string) string {
	file = filepath.Clean(file)
	base := filepath.Base(file)
	dir := filepath.Dir(file)
	if dir == "." {
		return base
	}
	parent := filepath.Base(dir)
	if parent == "." || parent == string(filepath.Separator) {
		return base
	}
	return filepath.Join(parent, base)
}

func GetCallerPosition(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "?:0"
	}
	return fmt.Sprintf("%s:%d", lastTwoPathParts(file), line)
}
