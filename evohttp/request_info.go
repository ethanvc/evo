package evohttp

import (
	"context"
	"net/http"
)

type RequestInfo struct {
	Request       *http.Request
	Writer        ResponseWriter
	UrlParams     map[string]string
	PatternPath   string
	ParsedRequest any
}

func NewRequestInfo() *RequestInfo {
	info := &RequestInfo{
		UrlParams: make(map[string]string),
	}
	return info
}

func (info *RequestInfo) UrlParam(key string) string {
	return info.UrlParams[key]
}

func GetRequestInfo(c context.Context) *RequestInfo {
	v, _ := c.Value(contextKeyRequestInfo{}).(*RequestInfo)
	return v
}

type contextKeyRequestInfo struct{}
