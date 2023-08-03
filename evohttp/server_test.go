package evohttp

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"path"
	"testing"
	"time"
)

func TestServer_Simple(t *testing.T) {
	svr := NewServer()
	url, httpSvr := startTestServer(svr)
	defer httpSvr.Shutdown(context.Background())

	test := func(c context.Context, req any, info *RequestInfo) (any, error) {
		return nil, nil
	}
	svr.POSTF("/test", test)
	st := NewSingleAttempt(http.MethodPost, url+"/test")
	err := st.Invoke(context.Background(), nil, nil)
	require.NoError(t, err)
}

func TestServer_GetJsonEcho(t *testing.T) {
	svr := NewServer()
	url, httpSvr := startTestServer(svr)
	defer httpSvr.Shutdown(context.Background())

	type Echo struct {
		Msg string
	}

	test := func(c context.Context, req *Echo) (*Echo, error) {
		return req, nil
	}
	svr.POST("/test", NewStdHandlerF(test))
	st := NewStdSingleAttempt(http.MethodPost, url+"/test")
	req := &Echo{
		Msg: "hello",
	}
	resp := &Echo{}
	err := st.Invoke(context.Background(), req, resp)
	require.NoError(t, err)
	require.Equal(t, req.Msg, resp.Msg)
}

func TestServer_Static(t *testing.T) {
	svr := NewServer()
	url, httpSvr := startTestServer(svr)
	defer httpSvr.Shutdown(context.Background())
	tmpDir, _ := os.MkdirTemp("", "evo_test")
	os.WriteFile(path.Join(tmpDir, "test.txt"), []byte("hello"), 0644)

	svr.Static("/static", tmpDir)
	st := NewSingleAttempt(http.MethodGet, url+"/static/test.txt")
	var content []byte
	err := st.Invoke(nil, nil, &content)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, st.Resp.StatusCode)
	require.Equal(t, "hello", string(content))
}

func TestServer_Root(t *testing.T) {
	svr := NewServer()
	url, httpSvr := startTestServer(svr)
	defer httpSvr.Shutdown(context.Background())
	tmpDir, _ := os.MkdirTemp("", "evo_test")
	os.WriteFile(path.Join(tmpDir, "index.html"), []byte("hello"), 0644)

	svr.Static("/", tmpDir)
	st := NewSingleAttempt(http.MethodGet, url)
	var content string
	err := st.Invoke(nil, nil, &content)
	require.NoError(t, err)
	require.Equal(t, "hello", content)
}

func startTestServer(handler http.Handler) (string, *http.Server) {
	addr := "127.0.0.1:9032"
	httpSvr := &http.Server{
		Handler: handler,
		Addr:    addr,
	}
	url := "http://" + addr
	go func() {
		err := httpSvr.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	var err error
	for i := 0; i < 3; i++ {
		var resp *http.Response
		resp, err = http.Get(url)
		if err != nil {
			time.Sleep(time.Millisecond)
			continue
		}
		resp.Body.Close()
		break
	}
	if err != nil {
		panic(err)
	}
	return url, httpSvr
}
