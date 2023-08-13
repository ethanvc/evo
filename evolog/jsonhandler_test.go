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
	h.Handle(WithLogContext(nil, "xxxx"), slog.NewRecord(time.Time{}, slog.LevelInfo, "hello", 0))
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
	h.Handle(nil, record)
	require.Equal(t,
		"{\"level\":\"INFO\",\"msg\":\"hello\",\"abc\":{\"Name\":\"\"},\"trace_id\":\"13eef5ee2424de958ed8010000695da4\"}\n",
		buf.String())
}
