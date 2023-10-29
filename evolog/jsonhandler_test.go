package evolog

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/require"
	"log/slog"
	"runtime"
	"testing"
	"time"
)

func Test_getFileSource(t *testing.T) {
	pc, _, _, _ := runtime.Caller(0)
	s := GetCallerLocation(pc)
	require.Equal(t, "evolog/jsonhandler_test.go:14", s)
}

func TestNewJsonHandler(t *testing.T) {
	var buf bytes.Buffer
	h := NewJsonHandler(NewJsonHandlerOpts().SetWriter(&buf).SetAddSource(false))
	h.Handle(WithLogContext(nil, &LogContextConfig{TraceId: "xxxx"}), slog.NewRecord(time.Time{}, slog.LevelInfo, "hello", 0))
	require.Equal(t, "{\"level\":\"INFO\",\"msg\":\"hello\",\"trace_id\":\"xxxx\"}\n", buf.String())
}

func TestJsonHandler_Ignore(t *testing.T) {
	type Abc struct {
		Name string `evolog:"ignore"`
	}
	var buf bytes.Buffer
	h := NewJsonHandler(NewJsonHandlerOpts().SetWriter(&buf).SetAddSource(false))
	abc := &Abc{Name: "Hello"}
	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "hello", 0)
	record.Add("abc", abc)
	c := WithLogContext(nil, &LogContextConfig{TraceId: "xx"})
	h.Handle(c, record)
	require.Equal(t,
		"{\"level\":\"INFO\",\"msg\":\"hello\",\"abc\":{\"Name\":\"\"},\"trace_id\":\"xx\"}\n",
		buf.String())
}

func Test_Error(t *testing.T) {
	var buf bytes.Buffer
	h := NewJsonHandler(NewJsonHandlerOpts().SetWriter(&buf).SetAddSource(false))
	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "hello", 0)
	record.AddAttrs(Error(errors.New("hello_error")))
	c := WithLogContext(nil, &LogContextConfig{TraceId: "xx"})
	h.Handle(c, record)
	require.Equal(t,
		"{\"level\":\"INFO\",\"msg\":\"hello\",\"err\":\"hello_error\",\"trace_id\":\"xx\"}\n",
		buf.String())
}
