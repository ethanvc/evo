package evolog

import (
	"context"
	"github.com/ethanvc/evo/evojson"
	"io"
	"log/slog"
)

type JsonHandler struct {
	h       *slog.JSONHandler
	encoder *Encoder
	lvl     slog.LevelVar
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
	if opts.Level != nil {
		h.lvl.Set(opts.Level.Level())
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
	h.h = slog.NewJSONHandler(w, opts)
	return h
}

func (h *JsonHandler) Handle(c context.Context, r slog.Record) error {
	lc := GetLogContext(c)
	r.AddAttrs(slog.String("trace_id", lc.GetTraceId()))
	return h.h.Handle(c, r)
}

func (h *JsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newH := &JsonHandler{
		encoder: h.encoder,
	}
	newH.lvl.Set(h.lvl.Level())
	newH.h = h.h.WithAttrs(attrs).(*slog.JSONHandler)
	return newH
}

func (h *JsonHandler) WithGroup(name string) slog.Handler {
	newH := &JsonHandler{
		encoder: h.encoder,
	}
	newH.lvl.Set(h.lvl.Level())
	newH.h = h.h.WithGroup(name).(*slog.JSONHandler)
	return newH
}

func (h *JsonHandler) Enabled(c context.Context, lvl slog.Level) bool {
	return h.lvl.Level() >= lvl
}

func (h *JsonHandler) SetLevel(lvl slog.Level) {
	h.lvl.Set(lvl)
}

func attrWrapper(encoder *Encoder, a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindAny:
		return slog.Any(a.Key, evojson.NewWrapper(encoder.configer.Load(), a.Value.Any()))
	}
	return a
}
