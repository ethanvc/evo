package evohttp

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethanvc/evo/base"
	"github.com/jinzhu/copier"
	"io"
	"net/http"
	"net/http/httptest"
)

func Example_paramInUrl() {
	svr := NewServer()
	type GetUserReq struct {
		UserId int64  `json:"user_id"`
		Name   string `json:"name"`
	}
	type GetUserResp struct {
		UserId int64  `json:"user_id"`
		Name   string `json:"name"`
	}
	h := func(c context.Context, req *GetUserReq) (resp *GetUserResp, err error) {
		resp = new(GetUserResp)
		copier.Copy(resp, req)
		return
	}
	svr.POST("/api/users/:user_id/:name", NewStdHandlerF(h))
	req := &GetUserReq{}
	httpRequest := httptest.NewRequest(http.MethodPost,
		"/api/users/123/jack", bytes.NewReader(base.AnyToJson(req)))
	httpRecoder := httptest.NewRecorder()
	svr.ServeHTTP(httpRecoder, httpRequest)
	result, _ := io.ReadAll(httpRecoder.Body)
	fmt.Print(string(result))
	// Output: {"code":0,"msg":"","event":"","data":{"user_id":123,"name":"jack"}}
}
