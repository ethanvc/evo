package evolog

import (
	"context"
	"github.com/ethanvc/evo/evojson"
	"io"
	"log/slog"
)

type JsonHandler struct {
	slog.JSONHandler
	encoder *Encoder
}

func NewJsonHandler(w io.Writer, encoder *Encoder, opts *slog.HandlerOptions) *JsonHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	if encoder == nil {
		encoder = DefaultEncoder()
	}
	h := &JsonHandler{
		encoder: encoder,
	}
	if h.encoder == nil {
		h.encoder = DefaultEncoder()
	}
	userReplaceAttr := opts.ReplaceAttr
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if userReplaceAttr != nil {
			a = userReplaceAttr(groups, a)
		}
		return attrWrapper(encoder, a)
	}
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

func attrWrapper(encoder *Encoder, a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindAny:
		return slog.Any(a.Key, evojson.NewWrapper(encoder.configer.Load(), a.Value.Any()))
	}
	return a
}
