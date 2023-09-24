package evohttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/ethanvc/evo/base"
	"io"
	"net/http"
	"strings"
)

type SingleAttempt struct {
	Request  *http.Request
	Response *http.Response
	Resp     any
	chain    base.Chain[*SingleAttempt]
}

func NewSingleAttempt(c context.Context, httpMethod, url string) *SingleAttempt {
	sa := &SingleAttempt{}
	sa.chain = defaultSingleAttemptInterceptors
	var err error
	sa.Request, err = http.NewRequestWithContext(c, httpMethod, url, nil)
	if err != nil {
		sa.Request, _ = http.NewRequestWithContext(c, httpMethod, "http://badurl_or_method_offered", nil)
	}
	return sa
}

func (sa *SingleAttempt) Do(req any, resp any) (err error) {
	sa.Resp = resp
	_, err = sa.chain.Do(sa.Request.Context(), req, sa)
	return
}

type SingleAttemptInterceptor struct {
}

func NewSingleAttemptInterceptor() *SingleAttemptInterceptor {
	return &SingleAttemptInterceptor{}
}

func (si *SingleAttemptInterceptor) Handle(c context.Context, req any, info *SingleAttempt, nexter base.Nexter[*SingleAttempt]) (resp any, err error) {
	resp, err = si.preHandle(c, req, info, nexter)
	if err != nil {
		return
	}
	info.Response, err = http.DefaultClient.Do(info.Request)
	if err != nil {
		return
	}
	resp, err = si.postHandle(c, req, info, nexter)
	if err != nil {
		return
	}
	if info.Response.StatusCode != http.StatusOK {
		return resp, ErrStatusNotOk
	}
	return
}

func (si *SingleAttemptInterceptor) preHandle(c context.Context, req any, info *SingleAttempt, nexter base.Nexter[*SingleAttempt]) (resp any, err error) {
	if req == nil || info.Request.Body != nil {
		return
	}
	contentType := info.Request.Header.Get("Content-Type")
	if len(contentType) == 0 {
		switch realReq := req.(type) {
		case string:
			info.Request.Body = io.NopCloser(strings.NewReader(realReq))
		case []byte:
			info.Request.Body = io.NopCloser(bytes.NewReader(realReq))
		default:
			buf, err := json.Marshal(req)
			if err != nil {
				return nil, err
			}
			info.Request.Body = io.NopCloser(bytes.NewReader(buf))
		}
		return
	}
	return
}

func (si *SingleAttemptInterceptor) postHandle(c context.Context, req any, info *SingleAttempt, nexter base.Nexter[*SingleAttempt]) (resp any, err error) {
	if info.Resp == nil {
		return
	}
	buf, err := io.ReadAll(info.Response.Body)
	info.Response.Body.Close()
	if err != nil {
		return
	}
	contentType := info.Response.Header.Get("Content-Type")
	switch mimeType := GetMimeType(contentType); mimeType {
	case "application/json":
		err = json.Unmarshal(buf, info.Resp)
		if err != nil {
			return
		}
	case "text/plain", "text/html":
		switch realResp := info.Resp.(type) {
		case *string:
			*realResp = string(buf)
		case *[]byte:
			*realResp = buf
		}
	}
	return info.Resp, nil
}

func GetMimeType(contentType string) string {
	i := 0
	for ; i < len(contentType); i++ {
		if contentType[i] == ';' {
			break
		}
	}
	return contentType[0:i]
}

var ErrStatusNotOk = errors.New("httpclient: status code not ok")

var defaultSingleAttemptInterceptors = []base.Interceptor[*SingleAttempt]{NewSingleAttemptInterceptor()}
