package xbufio

import (
	"bufio"
	"io"
)

type Reader struct {
	bufio.Reader
	rd io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		Reader: *bufio.NewReader(r),
		rd:     r,
	}
}

func (r *Reader) Close() error {
	if closer, ok := r.rd.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
