package evolog

import (
	"context"
	"fmt"
	"github.com/ethanvc/evo/evojson"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type JsonHandler struct {
	h    *slog.JSONHandler
	opts *JsonHandlerOpts
}

func NewJsonHandler(opts *JsonHandlerOpts) *JsonHandler {
	h := &JsonHandler{
		opts: opts,
	}
	h.init()
	return h
}

func (h *JsonHandler) init() {
	if h.opts == nil {
		h.opts = NewJsonHandlerOpts()
	}
	userReplaceAttr := h.opts.ReplaceAttr
	var realReplaceAttr func([]string, slog.Attr) slog.Attr
	if userReplaceAttr != nil {
		realReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			a = userReplaceAttr(groups, a)
			return attrWrapper(h.opts.Encoder, a)
		}
	} else {
		realReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			return attrWrapper(h.opts.Encoder, a)
		}
	}
	os.MkdirAll(filepath.Dir(h.opts.LogPath), 0770)
	var w io.Writer
	if h.opts.Writer != nil {
		w = h.opts.Writer
	}
	if w == nil {
		f, err := os.OpenFile(h.opts.LogPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
		}
		w = f
	}
	if w == nil {
		w = os.Stderr
	}
	// TODO make source beauty
	opts := slog.HandlerOptions{
		AddSource:   false,
		Level:       &h.opts.Level,
		ReplaceAttr: realReplaceAttr,
	}
	h.h = slog.NewJSONHandler(w, &opts)
	return
}

func (h *JsonHandler) Handle(c context.Context, r slog.Record) error {
	lc := GetLogContext(c)
	r.AddAttrs(slog.String("trace_id", lc.GetTraceId()))
	return h.h.Handle(c, r)
}

func (h *JsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *JsonHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *JsonHandler) Enabled(c context.Context, lvl slog.Level) bool {
	return lvl >= h.opts.Level.Level()
}

func (h *JsonHandler) SetLevel(lvl slog.Level) {
	h.opts.Level.Set(lvl)
}

type JsonHandlerOpts struct {
	AddSource   bool
	Level       slog.LevelVar
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
	LogPath     string
	Writer      io.Writer
	Encoder     *Encoder
}

func NewJsonHandlerOpts() *JsonHandlerOpts {
	wd, _ := os.Getwd()
	return &JsonHandlerOpts{
		AddSource: true,
		LogPath:   filepath.Join(wd, "log/evo.log"),
		Level:     slog.LevelVar{},
		Encoder:   DefaultEncoder(),
	}
}

func (opts *JsonHandlerOpts) SetWriter(w io.Writer) *JsonHandlerOpts {
	opts.Writer = w
	return opts
}

func attrWrapper(encoder *Encoder, a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindAny:
		return slog.Any(a.Key, evojson.NewWrapper(encoder.configer.Load(), a.Value.Any()))
	}
	return a
}
