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

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	resp, err := client.Get("https://www.baidu.com")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(string(body)), "baidu")
}

func Test_Connect_notFound(t *testing.T) {
	svr := &httpsvr.Server{}
	testSvr := httptest.NewServer(svr)
	defer testSvr.Close()

	req, err := http.NewRequest(http.MethodConnect, testSvr.URL+"/www.baidu.com:443", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func connectProxy(ctx context.Context, _ *httpsvr.Empty) (*httpsvr.Empty, error) {
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
	info.Hijacked = true
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
	return &httpsvr.Empty{}, nil
}
