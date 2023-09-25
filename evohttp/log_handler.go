package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/evolog"
	"log/slog"
	"time"
)

type LogHandler struct {
}

func NewLogHandler() *LogHandler {
	h := &LogHandler{}
	return h
}

func (h *LogHandler) Handle(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	resp, err = nexter.Next(c, req, info)
	evolog.LogRequest(c,
		&evolog.RequestLogInfo{
			Err:      err,
			Req:      info.ParsedRequest,
			Resp:     resp,
			Duration: time.Now().Sub(info.RequestTime),
		}, slog.Int("http_code", info.Writer.GetStatus()),
		slog.String("path", info.Request.URL.Path),
	)
	return
}
