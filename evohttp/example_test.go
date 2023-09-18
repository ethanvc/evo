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

func Example_a() {
	svr := NewServer()
	type GetUserReq struct {
		UserId int64 `json:"user_id"`
	}
	type GetUserResp struct {
		UserId int64 `json:"user_id"`
	}
	h := func(c context.Context, req *GetUserReq) (resp *GetUserResp, err error) {
		resp = new(GetUserResp)
		copier.Copy(resp, req)
		return
	}
	svr.POST("/api/users/:user_id", NewStdHandlerF(h))
	req := &GetUserReq{}
	httpRequest := httptest.NewRequest(http.MethodPost,
		"/api/users/123", bytes.NewReader(base.AnyToJson(req)))
	httpRecoder := httptest.NewRecorder()
	svr.ServeHTTP(httpRecoder, httpRequest)
	result, _ := io.ReadAll(httpRecoder.Body)
	fmt.Print(string(result))
	// Output: {"code":0,"msg":"","event":"","data":{"user_id":123}}
}
