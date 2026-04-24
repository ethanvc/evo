package xobsgin

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Writer struct {
	gin.ResponseWriter
	maxSize    int
	StatusCode int
	buf        bytes.Buffer
	RealSize   int
}

func newWriter(w gin.ResponseWriter) *Writer {
	return &Writer{
		ResponseWriter: w,
		maxSize:        1024 * 1024,
	}
}

func (w *Writer) Write(data []byte) (int, error) {
	if w.StatusCode == 0 {
		w.StatusCode = http.StatusOK
	}
	w.RealSize += len(data)
	if w.buf.Len() < w.maxSize {
		s := min(w.maxSize-w.buf.Len(), len(data))
		w.buf.Write(data[:s])
	}
	n, err := w.ResponseWriter.Write(data)
	return n, err
}

func (w *Writer) WriteHeader(status int) {
	if w.StatusCode < http.StatusOK {
		w.StatusCode = status
	}
	w.ResponseWriter.WriteHeader(status)
}
