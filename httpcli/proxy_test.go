package httpcli

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_CustomProxy(t *testing.T) {
	// example to test choice proxy dynamically.
	ctx := context.Background()
	target := ""
	trace := &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			target = hostPort
		},
	}
	transport := http.DefaultTransport.(*http.Transport)
	transport = transport.Clone()
	cli := &Client{
		DefaultClient: &http.Client{
			Transport: transport,
		},
	}
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		u, err := url.Parse("http://www.abc.com")
		if err != nil {
			return nil, err
		}
		return u, nil
	}
	ctx = httptrace.WithClientTrace(ctx, trace)
	err := cli.Do(ctx, "http://www.xx.com", "hello", nil, nil)
	_ = err
	require.Equal(t, "www.abc.com:80", target)
}
