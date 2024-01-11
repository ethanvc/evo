package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/plog"
	"log/slog"
)

type LogHandler struct {
	rl *plog.RequestLogger
}

func NewLogHandler() *LogHandler {
	h := &LogHandler{}
	return h
}

func (h *LogHandler) SetRequestLogger(rl *plog.RequestLogger) {
	h.rl = rl
}

func (h *LogHandler) Handle(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	c = plog.WithLogContext(c, &plog.LogContextConfig{
		Method:  info.PatternPath,
		TraceId: info.Request.Header.Get("x-trace-id"),
	})
	resp, err = nexter.Next(c, req, info)
	h.logger().Log(c, err, info.ParsedRequest, resp,
		slog.Int("http_code", info.Writer.GetStatus()),
		slog.String("path", info.Request.URL.Path),
	)
	return
}

func (h *LogHandler) logger() *plog.RequestLogger {
	if h.rl != nil {
		return h.rl
	} else {
		return plog.DefaultRequestLogger()
	}
}
