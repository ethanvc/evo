package httpcli

import (
	"context"
	"github.com/ethanvc/evo/base"
	"net/http"
)

type TransportHandler struct {
	cli *http.Client
}

func (h *TransportHandler) Handle(c context.Context, req any, info *SingleAttempt, nexter base.Nexter[*SingleAttempt]) (resp any, err error) {
	return
}
