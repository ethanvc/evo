package httpsvr

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ethanvc/study/golangproj/httpcli"
	"github.com/stretchr/testify/require"
)

func Test_HttpSvr_Basic(t *testing.T) {
	ctx := context.Background()
	svr := &Server{}
	testSvr := httptest.NewServer(svr)
	defer testSvr.Close()
	svr.Register("/api/*path", func(ctx context.Context, _ *Empty) (*Empty, error) {
		info := GetCallInfo(ctx)
		info.RespHeader.Set("Access-Control-Allow-Credentials", "true")
		info.RespHeader.Set("Access-Control-Allow-Headers", "Accept, Content-Type")
		info.RespHeader.Set("Access-Control-Allow-Methods", "POST, GET")
		// allow all origin, THIS IS DANGEROUS
		info.RespHeader.Set("Access-Control-Allow-Origin", "*")
		return &Empty{}, nil
	}, http.MethodOptions)
	svr.Register("/api/echo_body", func(ctx context.Context, req *string) (*string, error) {
		return req, nil
	}, http.MethodPost)
	svr.Register("/api/get", func(ctx context.Context, req *Empty) (*string, error) {
		resp := "/api/get"
		return &resp, nil
	}, http.MethodGet)
	svr.Register("/api/return_cookie", func(ctx context.Context, req *Empty) (*Empty, error) {
		info := GetCallInfo(ctx)
		http.SetCookie(info.Writer, &http.Cookie{
			Name:  "access_token",
			Value: "xxx",
		})
		return &Empty{}, nil
	}, http.MethodGet)

	opts := &httpcli.Options{}
	{
		opts.Method = http.MethodGet
		err := httpcli.Do(ctx, testSvr.URL+"/api/not_found", nil, nil, opts)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, opts.StatusCode)
	}
	{
		opts.Method = http.MethodGet
		err := httpcli.Do(ctx, testSvr.URL+"/api/return_cookie", nil, nil, opts)
		require.NoError(t, err)
		require.Equal(t, "access_token=xxx", opts.RespHeader.Get("Set-Cookie"))
	}
	{
		opts.Method = http.MethodGet
		resp, err := httpcli.DoType[string](ctx, testSvr.URL+"/api/get", nil, opts)
		require.NoError(t, err)
		require.Equal(t, "/api/get", *resp)
	}
	{
		opts.Method = http.MethodOptions
		err := httpcli.Do(ctx, testSvr.URL+"/api/test", nil, nil, opts)
		require.NoError(t, err)
		require.Equal(t, "true", opts.RespHeader.Get("Access-Control-Allow-Credentials"))
	}
	{
		resp, err := httpcli.DoType[string](ctx, testSvr.URL+"/api/echo_body", "hello", nil)
		require.NoError(t, err)
		require.Equal(t, "hello", *resp)
	}
}

func Test_HandlerCall(t *testing.T) {
	h := NewHandler(func(ctx context.Context, req *string) (*string, error) {
		return req, nil
	})
	req := "hello"
	resp, err := h.call(context.Background(), &req)
	require.NoError(t, err)
	require.Equal(t, "hello", *resp.(*string))

	// call with req=nil
	resp, err = h.call(context.Background(), nil)
	require.Equal(t, "req must not nil when call handler", err.Error())
}

func Test_validateAndParseFunc(t *testing.T) {
	f := func(context.Context, *any) (*any, error) {
		return nil, nil
	}
	reqType, err := validateAndParseFunc(f)
	require.NoError(t, err)
	require.Equal(t, "*interface {}", reqType.String())
}

func init() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(jsonHandler))
}
