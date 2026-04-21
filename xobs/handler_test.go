package xobs

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopWriteCloser struct {
	bytes.Buffer
}

func (*nopWriteCloser) Close() error { return nil }

func TestJsonHandler_Handle(t *testing.T) {
	var writer nopWriteCloser
	handler := NewJsonHandler(&writer)
	ctx := WithObsContext(context.Background(), &ObsConfig{Handler: handler})
	Info(ctx, "Test")
	require.Equal(t, "2026-01-01T00:00:00Z|info|xobs/handler_test.go:23|1234567890:1234567890:1234567890|Test\n", writer.String())
	writer.Reset()
	Info(ctx, "Test", String("key", "value"))
	require.Equal(t, `2026-01-01T00:00:00Z|info|xobs/handler_test.go:26|1234567890:1234567890:1234567890|Test|{"key":"value"}`+"\n", writer.String())
	writer.Reset()
	type Abc struct {
		Name string
	}
	Info(ctx, "Test", Any("abc", &Abc{Name: "value"}))
	require.Equal(t, `2026-01-01T00:00:00Z|info|xobs/handler_test.go:32|1234567890:1234567890:1234567890|Test|{"abc":{"Name":"value"}}`+"\n", writer.String())
}

func TestDefaultLogLevel(t *testing.T) {
	orig := GetDefaultLogLevel()
	defer SetDefaultLogLevel(orig)

	var writer nopWriteCloser
	handler := NewJsonHandler(&writer)
	ctx := WithObsContext(context.Background(), &ObsConfig{Handler: handler})

	// Default level is Info, so Dbg should be filtered out.
	Info(ctx, "Visible")
	assert.NotEmpty(t, writer.String())
	writer.Reset()

	obsCtx := GetObsContext(ctx)
	assert.False(t, obsCtx.Enabled(LevelDbg))
	assert.True(t, obsCtx.Enabled(LevelInfo))

	// Raise default level to Err: Info should now be filtered out.
	SetDefaultLogLevel(LevelErr)
	writer.Reset()
	Info(ctx, "ShouldNotAppear")
	assert.Empty(t, writer.String())

	Err(ctx, "VisibleErr")
	assert.NotEmpty(t, writer.String())
	assert.Contains(t, writer.String(), "VisibleErr")

	// Lower default level to Dbg: everything should pass.
	SetDefaultLogLevel(LevelDbg)
	writer.Reset()
	assert.True(t, obsCtx.Enabled(LevelDbg))
}

func TestGetCallerPosition(t *testing.T) {
	pos := GetCallerPosition(0)
	assert.True(t, strings.HasPrefix(pos, "xobs/handler_test.go:"))
}
