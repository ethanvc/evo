package evohttp

import (
	"context"
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

func (h *LogHandler) HandleRequest(c context.Context, req any, info *RequestInfo) (resp any, err error) {
	resp, err = info.Next(c, req)
	evolog.LogRequest(c, &evolog.RequestLogInfo{
		Err:      err,
		Req:      info.ParsedRequest,
		Resp:     resp,
		Duration: time.Now().Sub(info.RequestTime),
	}, slog.Int("http_code", info.Writer.GetStatus()))
	return
}
