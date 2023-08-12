package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"google.golang.org/grpc/codes"
	"log/slog"
)

type recoverHandler struct {
}

func NewRecoverHandler() Handler {
	return &recoverHandler{}
}

func (h *recoverHandler) HandleRequest(c context.Context, req any, info *RequestInfo) (resp any, err error) {
	func() {
		panicked := true
		defer func() {
			if !panicked {
				return
			}
			r := recover()
			switch realR := r.(type) {
			case error:
				err = realR
			case *base.Status:
				err = realR.Err()
			default:
				slog.ErrorContext(c, "UnknownPanicErr", slog.Any("err", r))
				err = base.New(codes.Internal, "UnknownPanicErr").Err()
			}
		}()
		resp, err = info.Next(c, req)
		panicked = false
	}()
	return
}
