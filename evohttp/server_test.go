package evohttp

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestServer_Simple(t *testing.T) {
	svr := NewServer()
	test := func(c context.Context, req any, info *RequestInfo) (any, error) {
		return nil, nil
	}
	listenAddr, url := getNextListenAddr()
	svr.POSTF("/test", test)
	go func() {
		svr.Run(listenAddr)
	}()
	st := NewSingleAttempt(http.MethodPost, url+"/test")
	err := st.Invoke(context.Background(), nil, nil)
	require.NoError(t, err)
}

var nextPortNumber int32

func getNextListenAddr() (listenAddr string, url string) {
	const minPort = 15000
	const maxPort = 16000
	for {
		port := atomic.AddInt32(&nextPortNumber, 1)
		if port < minPort || port > maxPort {
			atomic.CompareAndSwapInt32(&nextPortNumber, port, minPort)
			continue
		}
		listenAddr = fmt.Sprintf("127.0.0.1:%d", port)
		url = "http://" + listenAddr
		return
	}
}
