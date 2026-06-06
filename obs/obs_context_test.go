package obs

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestObsContext_AccessLogReport_plainError(t *testing.T) {
	var logBuf bytes.Buffer
	handler := NewJsonHandler(&logBuf)
	reg := prometheus.NewRegistry()
	rep := NewReporter(reg)

	ctx, _ := WithSpan(context.Background(), &SpanConfig{
		Method: "TestAPI",
		ObsConfig: ObsConfig{
			Handler: handler,
			Level:   LevelInfo,
		},
	})
	oc := GetObsContext(ctx)
	oc.reporter = rep

	reqErr := errors.New("xxx")
	oc.AccessLogReport(ctx, reqErr, "req-body", "resp-body", []KV{{Key: "biz", Val: "b1"}})

	logLine := logBuf.String()
	require.Contains(t, logLine, `{"method":"TestAPI","tc":"1.75µs","err":"xxx","req":"req-body","resp":"resp-body","attris":{}}`)
}

func Test1(t *testing.T) {
	var myErr *Error
	var err error
	err = myErr
	myErr2, ok := errors.AsType[*Error](err)
	require.True(t, ok)
	require.Equal(t, myErr, myErr2)
}
