package httpsvrexamles

import (
	"context"
	"fmt"
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

// 非TLS代理测试, 通过本地 HTTP 服务器做代理，中转 HTTP 请求到百度
func Test_NonTLSProxy(t *testing.T) {
	svr := &httpsvr.Server{}
	svr.Register("/*path", nonTLSProxy, http.MethodGet, http.MethodPost)

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
	cliResp, err := cli.Do(context.Background(), "http://www.baidu.com", nil, &body, &httpcli.Options{
		Method: http.MethodGet,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, cliResp.Response.StatusCode)
	assert.Contains(t, strings.ToLower(string(body)), "baidu")
}

// 非TLS代理 handler, 仅支持简单的 HTTP 代理，不支持 CONNECT
func nonTLSProxy(ctx context.Context, _ *httpsvr.Empty) (*httpsvr.Nil, error) {
	info := httpsvr.GetCallInfo(ctx)
	if info.Request.Body == nil {
		info.Request.Body = http.NoBody
	}
	target := strings.TrimPrefix(info.PathParms.ByName("path"), "/")
	if target == "" {
		target = info.Request.Host
	}
	// 重新组装完整的URL, 因为HTTP代理请求的Path其实是完整的URL
	fullURL := info.Request.RequestURI
	if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
		fullURL = "http://" + target + info.Request.RequestURI
	}

	// 通过 httpcli 请求上游
	header := info.Request.Header.Clone()
	header.Del("Proxy-Connection")

	var reqBody any
	if info.Request.Body != nil {
		reqBody = info.Request.Body
	}

	upstreamCli := &httpcli.Client{
		Timeout: 20 * time.Second,
	}
	var body []byte
	cliResp, err := upstreamCli.Do(ctx, fullURL, reqBody, &body, &httpcli.Options{
		Method: info.Request.Method,
		Header: header,
	})
	if err != nil {
		info.Writer.WriteHeader(http.StatusBadGateway)
		return nil, fmt.Errorf("fetch upstream: %w", err)
	}

	// 复制响应头
	for k, v := range cliResp.Response.Header {
		for _, vv := range v {
			info.Writer.Header().Add(k, vv)
		}
	}
	info.Writer.WriteHeader(cliResp.Response.StatusCode)
	_, err = info.Writer.Write(body)
	if err != nil {
		return nil, fmt.Errorf("copy resp body: %w", err)
	}

	return &httpsvr.Nil{}, nil
}
