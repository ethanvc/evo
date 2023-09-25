package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"net/http"
)

type rawHandlerWrapper struct {
	rawH http.HandlerFunc
}

func NewRawHandler(h func(w http.ResponseWriter, req *http.Request)) Handler {
	return &rawHandlerWrapper{
		rawH: h,
	}
}

func (h *rawHandlerWrapper) Handle(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	newHttpReq := info.Request.WithContext(c)
	h.rawH(&info.Writer, newHttpReq)
	return
}
