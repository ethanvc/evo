package evohttp

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestServer_Simple(t *testing.T) {
	svr := NewServer()
	listenAddr, url := getTestListenAddr()
	httpSvr := &http.Server{
		Handler: svr,
		Addr:    listenAddr,
	}
	go func() {
		httpSvr.ListenAndServe()
	}()
	defer httpSvr.Shutdown(context.Background())
	test := func(c context.Context, req any, info *RequestInfo) (any, error) {
		return nil, nil
	}
	svr.POSTF("/test", test)
	st := NewSingleAttempt(http.MethodPost, url+"/test")
	err := st.Invoke(context.Background(), nil, nil)
	require.NoError(t, err)
}

func getTestListenAddr() (listenAddr string, url string) {
	return "127.0.0.1:9032", "http://127.0.0.1:9032"
}
