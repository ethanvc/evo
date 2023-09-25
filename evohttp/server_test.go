package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"
)

func TestServer_Simple(t *testing.T) {
	svr := NewServer()
	url, httpSvr := startTestServer(svr)
	defer httpSvr.Shutdown(context.Background())

	test := func(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (any, error) {
		info.Writer.WriteHeader(http.StatusOK)
		return nil, nil
	}
	svr.POSTF("/test", test)
	st := NewSingleAttempt(context.Background(), http.MethodPost, url+"/test")
	err := st.Do(nil, nil)
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
	st := NewSingleAttempt(context.Background(), http.MethodPost, url+"/test")
	req := &Echo{
		Msg: "hello",
	}
	resp := &HttpResp[Echo]{}
	err := st.Do(req, resp)
	require.NoError(t, err)
	require.Equal(t, req.Msg, resp.Data.Msg)
}

func TestServer_Static(t *testing.T) {
	svr := NewServer()
	url, httpSvr := startTestServer(svr)
	defer httpSvr.Shutdown(context.Background())
	tmpDir, _ := os.MkdirTemp("", "evo_test")
	os.WriteFile(path.Join(tmpDir, "test.txt"), []byte("hello"), 0644)

	svr.Static("/static", tmpDir)
	st := NewSingleAttempt(context.Background(), http.MethodGet, url+"/static/test.txt")
	var content []byte
	err := st.Do(nil, &content)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, st.Response.StatusCode)
	require.Equal(t, "hello", string(content))
}

func TestServer_Root(t *testing.T) {
	svr := NewServer()
	url, httpSvr := startTestServer(svr)
	defer httpSvr.Shutdown(context.Background())
	tmpDir, _ := os.MkdirTemp("", "evo_test")
	os.WriteFile(path.Join(tmpDir, "index.html"), []byte("hello"), 0644)

	svr.Static("/", tmpDir)
	st := NewSingleAttempt(context.Background(), http.MethodGet, url)
	var content string
	err := st.Do(nil, &content)
	require.NoError(t, err)
	require.Equal(t, "hello", content)
}

func Test_Default404(t *testing.T) {
	svr := NewServer()
	url, httpSvr := startTestServer(svr)
	defer httpSvr.Shutdown(context.Background())

	st := NewSingleAttempt(context.Background(), http.MethodGet, url+"/abc")
	var content string
	err := st.Do(nil, &content)
	require.Equal(t, ErrStatusNotOk, err)
	require.Equal(t, http.StatusNotFound, st.Response.StatusCode)
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

func Test_PanicRecover(t *testing.T) {
	svr := NewServer()
	h := func(c context.Context, req *int) (resp *int, err error) {
		panic(base.New(codes.NotFound, ""))
		return
	}
	svr.GET("/abc", NewStdHandlerF(h))
	httpReq := httptest.NewRequest(http.MethodGet, "/abc", nil)
	recorder := httptest.NewRecorder()
	svr.ServeHTTP(recorder, httpReq)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "{\"code\":5,\"msg\":\"\",\"event\":\"\",\"data\":null}", recorder.Body.String())
}

func TestPanic2(t *testing.T) {
	svr := NewServer()
	h := func(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
		panic(base.New(codes.NotFound, ""))
		return
	}
	svr.GET("/abc", HandlerFunc(h))
	httpReq := httptest.NewRequest(http.MethodGet, "/abc", nil)
	recorder := httptest.NewRecorder()
	svr.ServeHTTP(recorder, httpReq)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, "internal server error", recorder.Body.String())
}
