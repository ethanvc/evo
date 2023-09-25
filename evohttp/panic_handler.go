package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"google.golang.org/grpc/codes"
	"log/slog"
)

type PanicHandler struct {
}

func NewPanicHandler() *PanicHandler {
	h := &PanicHandler{}
	return h
}

func (h *PanicHandler) Handle(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	func() {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			switch realT := r.(type) {
			case error:
				err = realT
			default:
				slog.ErrorContext(c, "UnknownPanic", slog.Any("err", err))
				err = base.New(codes.Internal, "ServerPanicked").Err()
			}
		}()
		resp, err = nexter.Next(c, req, info)
	}()
	return
}
