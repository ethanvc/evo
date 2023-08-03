package evolog

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"testing"
	"time"
)

func TestNewJsonHandler(t *testing.T) {
	var buf bytes.Buffer
	h := NewJsonHandler(&buf, nil)
	h.Handle(WithLogContext(nil, "xxxx"), slog.NewRecord(time.Time{}, slog.LevelInfo, "hello", 0))
	require.Equal(t, "{\"level\":\"INFO\",\"msg\":\"hello\",\"trace_id\":\"xxxx\"}\n", buf.String())
}
