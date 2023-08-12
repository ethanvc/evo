package evolog

import (
	"context"
	"io"
	"log/slog"
)

type JsonHandler struct {
	slog.JSONHandler
}

func NewJsonHandler(w io.Writer, opts *slog.HandlerOptions) *JsonHandler {
	h := &JsonHandler{}
	h.JSONHandler = *slog.NewJSONHandler(w, opts)
	return h
}

func (h *JsonHandler) Handle(c context.Context, r slog.Record) error {
	lc := GetLogContext(c)
	r.AddAttrs(slog.String("trace_id", lc.GetTraceId()))
	return h.JSONHandler.Handle(c, r)
}

func (h *JsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &JsonHandler{
		JSONHandler: *h.JSONHandler.WithAttrs(attrs).(*slog.JSONHandler),
	}
}

func (h *JsonHandler) WithGroup(name string) slog.Handler {
	return &JsonHandler{
		JSONHandler: *h.JSONHandler.WithGroup(name).(*slog.JSONHandler),
	}
}
