package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/evolog"
	"log/slog"
)

type LogHandler struct {
	rl *evolog.RequestLogger
}

func NewLogHandler() *LogHandler {
	h := &LogHandler{}
	return h
}

func (h *LogHandler) SetRequestLogger(rl *evolog.RequestLogger) {
	h.rl = rl
}

func (h *LogHandler) Handle(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	c = evolog.WithLogContext(c, &evolog.LogContextConfig{
		Method:  info.PatternPath,
		TraceId: info.Request.Header.Get("x-trace-id"),
	})
	resp, err = nexter.Next(c, req, info)
	h.logger().Log(c,
		&evolog.RequestLogInfo{
			Err:  err,
			Req:  info.ParsedRequest,
			Resp: resp,
		}, slog.Int("http_code", info.Writer.GetStatus()),
		slog.String("path", info.Request.URL.Path),
	)
	return
}

func (h *LogHandler) logger() *evolog.RequestLogger {
	if h.rl != nil {
		return h.rl
	} else {
		return evolog.DefaultRequestLogger()
	}
}
