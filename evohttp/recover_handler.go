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

func (h *recoverHandler) Handle(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	func() {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
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
		resp, err = nexter.Next(c, req, info)
	}()
	return
}
