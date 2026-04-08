package httpsvr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

var DefaultLogger Logger = &LoggerImpl{}

var DefaultSerializer Serializer = &JsonSerializer{}

var DefaultNotFoundHandler = NewHandler(func(ctx context.Context, empty *Empty) (*Empty, error) {
	info := GetCallInfo(ctx)
	info.StatusCode = http.StatusNotFound
	return &Empty{}, nil
})

type LoggerImpl struct{}

func (l *LoggerImpl) Start(ctx context.Context, info *CallInfo) context.Context {
	return ctx
}

func (l *LoggerImpl) End(ctx context.Context, err error, req, resp any, info *CallInfo) {
	lvl := slog.LevelInfo
	if err != nil {
		lvl = slog.LevelError
	}
	slog.Log(ctx, lvl, "REQ_END", "method", info.Request.Method,
		"pattern", info.Pattern,
		"url", info.Request.URL.String(),
		"err", err, "status_code", info.StatusCode,
		"req", req, "resp", resp)
}

func (l *LoggerImpl) Log(ctx context.Context, lvl slog.Level, event string, args ...any) {
	slog.Log(ctx, lvl, event, args...)
}

type JsonSerializer struct {
}

func (j *JsonSerializer) Marshal(ctx context.Context, err error, v any, info *CallInfo) (io.ReadCloser, error) {
	buf, newErr := json.Marshal(v)
	if newErr != nil {
		return nil, newErr
	}
	info.ResponseBody = buf
	info.RespHeader.Set("content-type", "application/json")
	return io.NopCloser(bytes.NewReader(buf)), nil
}
func (j *JsonSerializer) GetStatusCode(ctx context.Context, err error) int {
	return 0
}
func (j *JsonSerializer) Unmarshal(ctx context.Context, v any, info *CallInfo) error {
	err := json.Unmarshal(info.RequestBody, v)
	return err
}
