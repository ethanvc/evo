package evohttp

import "net/http"

type ResponseWriter struct {
	w      http.ResponseWriter
	status int
}

func (w *ResponseWriter) Reset(rawW http.ResponseWriter) {
	w.w = rawW
	w.status = 0
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.w.Write(p)
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.w.WriteHeader(statusCode)
}

func (w *ResponseWriter) GetStatus() int {
	return w.status
}
