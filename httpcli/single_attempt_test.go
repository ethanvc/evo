package httpcli

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptrace"
	"testing"
)

func TestSingleAttempt_Basic(t *testing.T) {
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			fmt.Printf("%s\n", info.Conn.RemoteAddr().String())
		},
	}
	c := httptrace.WithClientTrace(context.Background(), trace)
	httpReq, _ := http.NewRequestWithContext(c, http.MethodGet, "https://www.baidu.com", nil)
	httpResp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	_ = httpResp
}
