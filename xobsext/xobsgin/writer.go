package xobsgin

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Writer struct {
	gin.ResponseWriter
	maxSize    int
	statusCode int
	buf        bytes.Buffer
	realSize   int
}

func newWriter(w gin.ResponseWriter) *Writer {
	return &Writer{
		ResponseWriter: w,
		maxSize:        1024 * 1024,
	}
}

func (w *Writer) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	w.realSize += len(data)
	if w.buf.Len() < w.maxSize {
		s := min(w.maxSize-w.buf.Len(), len(data))
		w.buf.Write(data[:s])
	}
	n, err := w.ResponseWriter.Write(data)
	return n, err
}

func (w *Writer) WriteHeader(status int) {
	if w.statusCode < http.StatusOK {
		w.statusCode = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *Writer) Status() int {
	return w.statusCode
}

func (w *Writer) GetRealSize() int {
	return w.realSize
}

func (w *Writer) Bytes() []byte {
	return w.buf.Bytes()
}

func (w *Writer) String() string {
	return w.buf.String()
}

func (w *Writer) Truncated() bool {
	return w.realSize != w.buf.Len()
}
