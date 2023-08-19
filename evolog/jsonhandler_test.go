package evolog

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestNewJsonHandler(t *testing.T) {
	var buf bytes.Buffer
	h := NewJsonHandler(&buf, nil, nil)
	h.Handle(WithLogContext(nil, &LogContextConfig{TraceId: "xxxx"}), slog.NewRecord(time.Time{}, slog.LevelInfo, "hello", 0))
	require.Equal(t, "{\"level\":\"INFO\",\"msg\":\"hello\",\"trace_id\":\"xxxx\"}\n", buf.String())
}

func TestJsonHandler_Ignore(t *testing.T) {
	type Abc struct {
		Name string `evolog:"ignore"`
	}
	var buf bytes.Buffer
	h := NewJsonHandler(&buf, nil, nil)
	abc := &Abc{Name: "Hello"}
	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "hello", 0)
	record.Add("abc", abc)
	c := WithLogContext(nil, &LogContextConfig{TraceId: "xx"})
	h.Handle(c, record)
	require.Equal(t,
		"{\"level\":\"INFO\",\"msg\":\"hello\",\"abc\":{\"Name\":\"\"},\"trace_id\":\"xx\"}\n",
		buf.String())
}
