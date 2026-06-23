package httpsvr

import (
	"context"
	"net/http"
)

var defaultNotFoundHandler = NewHandler(NotFoundHandler)

func NotFoundHandler(ctx context.Context, empty *Empty) (*Empty, error) {
	info := GetCallInfo(ctx)
	info.StatusCode = http.StatusNotFound
	return &Empty{}, nil
}
