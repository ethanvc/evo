package httpsvrexamles

import (
	"context"
	"fmt"
	"io"
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
func nonTLSProxy(ctx context.Context, _ *httpsvr.Nil) (*httpsvr.Nil, error) {
	info := httpsvr.GetCallInfo(ctx)
	target := strings.TrimPrefix(info.PathParms.ByName("path"), "/")
	if target == "" {
		target = info.Request.Host
	}
	// 重新组装完整的URL, 因为HTTP代理请求的Path其实是完整的URL
	fullURL := info.Request.RequestURI
	if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
		fullURL = "http://" + target + info.Request.RequestURI
	}

	// 构建 http.Request 给远程服务器
	req, err := http.NewRequestWithContext(ctx, info.Request.Method, fullURL, info.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	// 复制所有 header
	for k, v := range info.Request.Header {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	// 删除 Proxy 相关头部（防止影响下游）
	req.Header.Del("Proxy-Connection")

	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		info.Writer.WriteHeader(http.StatusBadGateway)
		return nil, fmt.Errorf("fetch upstream: %w", err)
	}
	defer resp.Body.Close()

	// 复制响应头
	for k, v := range resp.Header {
		for _, vv := range v {
			info.Writer.Header().Add(k, vv)
		}
	}
	info.Writer.WriteHeader(resp.StatusCode)
	_, err = io.Copy(info.Writer, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("copy resp body: %w", err)
	}

	return &httpsvr.Nil{}, nil
}
