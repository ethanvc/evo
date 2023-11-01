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
	h := &LogHandler{
		rl: evolog.NewRequestLogger(),
	}
	return h
}

func (h *LogHandler) Handle(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	c = evolog.WithLogContext(c, &evolog.LogContextConfig{
		Method:  info.PatternPath,
		TraceId: info.Request.Header.Get("x-trace-id"),
	})
	resp, err = nexter.Next(c, req, info)
	h.rl.Log(c,
		&evolog.RequestLogInfo{
			Err:  err,
			Req:  info.ParsedRequest,
			Resp: resp,
		}, slog.Int("http_code", info.Writer.GetStatus()),
		slog.String("path", info.Request.URL.Path),
	)
	return
}
