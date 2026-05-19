// Copyright 2026 The evo Authors. All rights reserved.

package httpmux

import "net/http"

type Handler = http.Handler
type HandlerFunc = http.HandlerFunc
type Request = http.Request
type ResponseWriter = http.ResponseWriter

const (
	StatusBadRequest        = http.StatusBadRequest
	StatusMethodNotAllowed  = http.StatusMethodNotAllowed
	StatusMovedPermanently  = http.StatusMovedPermanently
	StatusTemporaryRedirect = http.StatusTemporaryRedirect
)

func Error(w ResponseWriter, error string, code int) {
	http.Error(w, error, code)
}

func NotFoundHandler() Handler {
	return http.NotFoundHandler()
}

func RedirectHandler(url string, code int) Handler {
	return http.RedirectHandler(url, code)
}

func StatusText(code int) string {
	return http.StatusText(code)
}
