package evohttp

import (
	"context"
	"net/http"
	"time"
)

type RequestInfo struct {
	Request       *http.Request
	Writer        ResponseWriter
	params        map[string]string
	handlers      HandlerChain
	PatternPath   string
	handlerIndex  int
	ParsedRequest any

	RequestTime time.Time
	FinishTime  time.Time
}

func NewRequestInfo() *RequestInfo {
	info := &RequestInfo{
		params: make(map[string]string),
	}
	return info
}

func (info *RequestInfo) Next(c context.Context, req any) (any, error) {
	index := info.handlerIndex
	info.handlerIndex++
	if index >= len(info.handlers) {
		return nil, nil
	}
	return info.handlers[index].HandleRequest(c, req, info)
}

func (info *RequestInfo) Handler() Handler {
	l := len(info.handlers)
	if l == 0 {
		return nil
	}
	return info.handlers[l-1]
}

func (info *RequestInfo) ResetHandlers(handlers HandlerChain) {
	info.handlers = handlers
	info.handlerIndex = 0
}

func (info *RequestInfo) UrlParam(key string) string {
	return info.params[key]
}

func GetRequestInfo(c context.Context) *RequestInfo {
	v, _ := c.Value(contextKeyRequestInfo{}).(*RequestInfo)
	return v
}

type contextKeyRequestInfo struct{}
