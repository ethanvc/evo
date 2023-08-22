package evohttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"io"
	"net/http"
)

type SingleAttempt struct {
	Req          *http.Request
	Resp         *http.Response
	interceptors []AttemptInterceptor
	index        int
}

func NewSingleAttempt(httpMethod, url string) *SingleAttempt {
	sa := &SingleAttempt{}
	var err error
	sa.Req, err = http.NewRequest(httpMethod, url, nil)
	if err != nil {
		sa.Req, _ = http.NewRequest(http.MethodGet, "http://error.when.new.request", nil)
		return sa
	}
	return sa
}

func NewJsonSingleAttempt(httpMethod, url string) *SingleAttempt {
	sa := NewSingleAttempt(httpMethod, url)
	sa.AppendInterceptorsF(AttemptCodecJson)
	return sa
}

func NewStdSingleAttempt(httpMethod, url string) *SingleAttempt {
	sa := NewSingleAttempt(httpMethod, url)
	sa.AppendInterceptorsF(AttemptStdCodecJson, AttemptCodecJson)
	return sa
}

func (sa *SingleAttempt) AppendInterceptors(interceptors ...AttemptInterceptor) {
	sa.interceptors = append(sa.interceptors, interceptors...)
}

func (sa *SingleAttempt) AppendInterceptorsF(interceptors ...AttemptInterceptorFunc) {
	var ints []AttemptInterceptor
	for _, v := range interceptors {
		ints = append(ints, v)
	}
	sa.AppendInterceptors(ints...)
}

func (sa *SingleAttempt) Next(c context.Context, req, resp any) error {
	index := sa.index
	sa.index++
	if index < len(sa.interceptors) {
		return sa.interceptors[index].HandleRequest(c, req, resp, sa)
	}
	if index == len(sa.interceptors) {
		return sa.do(c, req, resp)
	}
	return nil
}

func (sa *SingleAttempt) do(c context.Context, req, resp any) error {
	var err error
	if req != nil && sa.Req.Body == nil {
		switch v := req.(type) {
		case []byte:
			sa.Req.Body = io.NopCloser(bytes.NewReader(v))
		case io.ReadCloser:
			sa.Req.Body = v
		case io.Reader:
			sa.Req.Body = io.NopCloser(v)
		}
	}
	sa.Resp, err = http.DefaultClient.Do(sa.Req)
	if err != nil {
		return err
	}
	if resp != nil {
		switch v := resp.(type) {
		case *[]byte:
			buf, err := io.ReadAll(sa.Resp.Body)
			sa.Resp.Body.Close()
			if err != nil {
				return err
			}
			*v = buf
		case *string:
			buf, err := io.ReadAll(sa.Resp.Body)
			sa.Resp.Body.Close()
			if err != nil {
				return err
			}
			*v = string(buf)
		}
	}
	if sa.Resp.StatusCode/100 != 2 {
		return ErrStatusCodeNotOk
	}
	return nil
}

func (sa *SingleAttempt) Invoke(c context.Context, req, resp any) error {
	return sa.Next(c, req, resp)
}

type AttemptInterceptor interface {
	HandleRequest(c context.Context, req, resp any, sa *SingleAttempt) error
}

type AttemptInterceptorFunc func(c context.Context, req, resp any, sa *SingleAttempt) error

func (f AttemptInterceptorFunc) HandleRequest(c context.Context, req, resp any, sa *SingleAttempt) error {
	return f(c, req, resp, sa)
}

func AttemptCodecJson(c context.Context, req, resp any, sa *SingleAttempt) error {
	var err error
	if req != nil && sa.Req.Body == nil {
		buf, err := json.Marshal(req)
		if err != nil {
			return err
		}
		sa.Req.Body = io.NopCloser(bytes.NewReader(buf))
		sa.Req.Header.Set("Content-Type", "application/json")
	}
	err = sa.Next(c, req, resp)
	if err != nil {
		return err
	}
	buf, err := io.ReadAll(sa.Resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, resp)
	if err != nil {
		return err
	}
	if sa.Resp.StatusCode != http.StatusOK {
		return ErrStatusNotOk
	}
	return nil
}

func AttemptStdCodecJson(c context.Context, req, resp any, sa *SingleAttempt) error {
	var realResp HttpResp
	realResp.Data = resp
	err := sa.Next(c, req, &realResp)
	if err != nil {
		return err
	}
	if realResp.Code == codes.OK {
		return nil
	}
	realErr := &Error{
		Code:  realResp.Code,
		Msg:   realResp.Msg,
		Event: realResp.Event,
	}
	return realErr
}

var ErrStatusNotOk = fmt.Errorf("ErrStatusNotOk")

type HttpResp struct {
	Code  codes.Code `json:"code"`
	Msg   string     `json:"msg"`
	Event string     `json:"event"`
	Data  any        `json:"data"`
}

type Error struct {
	Code  codes.Code `json:"code"`
	Msg   string     `json:"msg"`
	Event string     `json:"event"`
}

func (e *Error) Error() string {
	buf, _ := json.Marshal(e)
	return string(buf)
}

func Code(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	realErr, _ := err.(*Error)
	if realErr != nil {
		return realErr.Code
	} else {
		return codes.Internal
	}
}

var ErrStatusCodeNotOk = errors.New("StatusCodeNotOk")
