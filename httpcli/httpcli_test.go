package httpcli

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_SpecialType(t *testing.T) {
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
	cliResp, err := Do(ctx, svr.URL, nil, nil, opts)
	require.NoError(t, err)
	require.Zero(t, len(cliResp.Body))

	var tmpStr string
	cliResp, err = Do(ctx, svr.URL, "TEST", &tmpStr, nil)
	require.NoError(t, err)
	require.Equal(t, "TEST", tmpStr)

	cliResp, err = Do(ctx, svr.URL, `{"a":"3""}`, &tmpStr, nil)
	require.NoError(t, err)
	require.Equal(t, `{"a":"3""}`, tmpStr)
}

func Test_Manul(t *testing.T) {
	ctx := context.Background()
	body := ""
	cliResp, err := Do(ctx, "https://chat.baidu.com/aichat/api/messages/list", nil, &body, nil)
	_ = err
	_ = cliResp
}
