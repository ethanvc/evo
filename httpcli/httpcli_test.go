package httpcli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_Do(t *testing.T) {
	ctx := context.Background()
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		_, _ = w.Write(body)
	}))
	defer svr.Close()

	// req is nil
	opts := &Options{}
	err := Do(ctx, svr.URL, nil, nil, opts)
	require.NoError(t, err)
	require.Zero(t, len(opts.RespBody))
	var tmpStr string
	err = Do(ctx, svr.URL, "TEST", &tmpStr, nil)
	require.NoError(t, err)
	require.Equal(t, "TEST", tmpStr)
	var tmpAny map[string]string
	err = Do(ctx, svr.URL, `{"a":"3""}`, &tmpAny, nil)
	require.Equal(t, `unmarshal error: invalid character '"' after object key:value pair. body is {"a":"3""}`, err.Error())

	tmpAny = nil
	err = Do(ctx, svr.URL, `{"a":"3"}`, &tmpAny, nil)
	require.NoError(t, err)
	require.Equal(t, "3", tmpAny["a"])

	err = Do(ctx, svr.URL, bytes.NewBuffer([]byte("hello")), &tmpStr, nil)
	require.NoError(t, err)
	require.Equal(t, "hello", tmpStr)
}
