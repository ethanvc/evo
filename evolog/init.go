package evolog

import (
	"io"
	"os"
	"path/filepath"

	"golang.org/x/exp/slog"
)

func init() {
	initTraceIdSeed()
	globalLogContext = &LogContext{
		traceId: NewTraceId(),
		method:  "Global",
	}
	initDefaultLog()
}

func initDefaultLog() {
	filePath := "./log/evo.log"
	os.MkdirAll(filepath.Dir(filePath), 0770)
	var w io.WriteCloser
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		w = os.Stderr
	} else {
		w = f
	}
	h := NewJsonHandler(w, nil)
	l := slog.New(h)
	slog.SetDefault(l)
}
