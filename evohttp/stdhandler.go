package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"google.golang.org/grpc/codes"
)

type StdHandler struct {
	newReq      func() any
	realHandler func(context.Context, any) (any, error)
}

func NewStdHandlerF[Req any, Resp any](f func(c context.Context, req *Req) (*Resp, error)) Handler {
	h := &StdHandler{}
	h.newReq = func() any {
		return new(Req)
	}
	h.realHandler = func(c context.Context, req any) (any, error) {
		var realReq *Req
		if req != nil {
			realReq = (req).(*Req)
		}
		realResp, err := f(c, realReq)
		return realResp, err
	}
	return h
}

func (h StdHandler) NewReq() any {
	if h.newReq == nil {
		return nil
	}
	return h.newReq()
}

func (h StdHandler) HandleRequest(c context.Context, req any, info *RequestInfo) (any, error) {
	if h.realHandler == nil {
		return nil, base.New(codes.Internal, "HandlerEmpty").Err()
	}
	return h.realHandler(c, req)
}

type EmptyRequest struct {
}

type EmptyResponse struct {
}

type HttpResp[T any] struct {
	Code  codes.Code `json:"code"`
	Msg   string     `json:"msg"`
	Event string     `json:"event"`
	Data  T          `json:"data"`
}
