package xobsgin

import (
	"bytes"
	"io"
)

type Reader struct {
	io.ReadCloser
	maxSize  int
	buf      bytes.Buffer
	realSize int
}

func newReader(r io.ReadCloser) *Reader {
	return &Reader{
		ReadCloser: r,
		maxSize:    1024 * 1024,
	}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.ReadCloser.Read(p)
	s := min(n, r.maxSize-r.buf.Len())
	r.buf.Write(p[:s])
	return n, err
}

func (r *Reader) Bytes() []byte {
	return r.buf.Bytes()
}
