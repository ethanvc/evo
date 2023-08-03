package evolog

import (
	"golang.org/x/exp/slog"
	"io"
	"os"
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
	f, _ := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	var w io.Writer
	w = f
	if w == nil {
		w = os.Stdout
	}
	h := NewJsonHandler(w, nil)
	l := slog.New(h)
	slog.SetDefault(l)
}

func OnProcessExit() {

}
