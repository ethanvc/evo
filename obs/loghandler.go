package obs

import (
	"context"
	"io"
)

type LogHandler interface {
	Handle(ctx context.Context, item LogItem)
	Flush()
	io.Closer
}
