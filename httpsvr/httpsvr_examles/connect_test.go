package httpsvrexamles

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ethanvc/evo/httpcli"
	"github.com/ethanvc/evo/httpsvr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Connect(t *testing.T) {
	svr := &httpsvr.Server{}
	svr.Register("/*path", connectProxy, http.MethodConnect)

	testSvr := httptest.NewServer(svr)
	defer testSvr.Close()

	proxyURL, err := url.Parse(testSvr.URL)
	require.NoError(t, err)

	cli := &httpcli.Client{
		Timeout: 30 * time.Second,
		DefaultClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		},
	}

	var body []byte
	cliResp, err := cli.Do(context.Background(), "https://www.baidu.com", nil, &body, &httpcli.Options{
		Method: http.MethodGet,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, cliResp.Response.StatusCode)
	assert.Contains(t, strings.ToLower(string(body)), "baidu")
}

func connectProxy(ctx context.Context, _ *httpsvr.Empty) (*httpsvr.Nil, error) {
	info := httpsvr.GetCallInfo(ctx)
	target := strings.TrimPrefix(info.PathParms.ByName("path"), "/")
	if target == "" {
		target = info.Request.Host
	}
	if _, _, err := net.SplitHostPort(target); err != nil {
		return nil, fmt.Errorf("invalid connect target %q: %w", target, err)
	}

	hj, ok := info.Writer.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("response writer does not support hijack")
	}
	clientConn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	defer clientConn.Close()

	upstream, err := net.DialTimeout("tcp", target, 15*time.Second)
	if err != nil {
		_, _ = clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return nil, err
	}
	defer upstream.Close()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return nil, err
	}
	if rw != nil && rw.Reader.Buffered() > 0 {
		if _, err := io.Copy(upstream, rw.Reader); err != nil {
			return nil, err
		}
	}

	errc := make(chan error, 2)
	go func() { _, err := io.Copy(upstream, clientConn); errc <- err }()
	go func() { _, err := io.Copy(clientConn, upstream); errc <- err }()
	<-errc
	upstream.Close()
	<-errc
	return &httpsvr.Nil{}, nil
}
