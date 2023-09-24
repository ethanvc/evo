package evohttp

import (
	"context"
	"errors"
	"github.com/ethanvc/evo/base"
	"net/http"
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
	info.Response, err = http.DefaultClient.Do(info.Request)
	if err != nil {
		return
	}
	if info.Response.StatusCode != http.StatusOK {
		return resp, ErrStatusNotOk
	}
	return
}

var ErrStatusNotOk = errors.New("httpclient: status code not ok")

var defaultSingleAttemptInterceptors = []base.Interceptor[*SingleAttempt]{NewSingleAttemptInterceptor()}
