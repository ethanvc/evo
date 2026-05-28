package bevo

import "io"

type LimitWriter struct {
	w        io.Writer
	maxBytes int
	writen   int
}

func NewLimitWriter(w io.Writer, maxBytes int) *LimitWriter {
	if maxBytes == 0 {
		maxBytes = 1024 * 1024
	}
	return &LimitWriter{
		w:        w,
		maxBytes: maxBytes,
	}
}
