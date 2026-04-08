package httpcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var DefaultSerializer = &JsonSerializer{}

var sDefaultClient = &Client{}

func GetDefault() *Client {
	return sDefaultClient
}

func Do(ctx context.Context, url string, req, resp any, opts *Options) error {
	return GetDefault().Do(ctx, url, req, resp, opts)
}

func DoType[Resp any](ctx context.Context, url string, req any, opts *Options) (*Resp, error) {
	var resp Resp
	err := Do(ctx, url, req, &resp, opts)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type Client struct {
	Serializer    Serializer
	Timeout       time.Duration
	Interceptors  []Interceptor
	DefaultClient *http.Client
}

func (cli *Client) Do(ctx context.Context, url string, req, resp any, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	} else {
		opts.StatusCode = 0
		opts.RespBody = nil
		opts.RespHeader = nil
	}
	ctx, cancel := cli.handleTimeout(ctx, opts.Timeout)
	if cancel != nil {
		defer cancel()
	}
	next := Next{
		interceptors: cli.Interceptors,
		handler:      cli.handle,
	}
	return next.Next(ctx, url, req, resp, opts)
}

func (cli *Client) handle(ctx context.Context, url string, req, resp any, opts *Options) error {
	contentType, reqBody, err := cli.marshal(ctx, req, opts)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, opts.GetMethod(), url, reqBody)
	if err != nil {
		return err
	}
	if len(opts.Header) > 0 {
		httpReq.Header = opts.Header
	}
	if contentType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	httpResp, err := cli.getHttpClient().Do(httpReq)
	if err != nil {
		return err
	}
	opts.StatusCode = httpResp.StatusCode
	opts.RespHeader = httpResp.Header
	err = cli.unmarshal(ctx, httpResp, resp, opts)
	if err != nil {
		return err
	}
	return nil
}

func (cli *Client) marshal(ctx context.Context, req any, opts *Options) (string, io.Reader, error) {
	if req == nil {
		return "", nil, nil
	}
	switch realReq := req.(type) {
	case string:
		return "", strings.NewReader(realReq), nil
	case []byte:
		return "", bytes.NewReader(realReq), nil
	case io.Reader:
		return "", realReq, nil
	}
	serializer := cli.getSerializer(opts)
	return serializer.Marshal(ctx, req, opts)
}

func (cli *Client) unmarshal(ctx context.Context, httpResp *http.Response, resp any, opts *Options) error {
	if readCloser, ok := resp.(*io.ReadCloser); ok {
		*readCloser = httpResp.Body
		return nil
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	opts.RespBody = body
	if resp == nil || len(body) == 0 {
		return nil
	}
	switch realV := resp.(type) {
	case *string:
		*realV = string(body)
		return nil
	case *[]byte:
		*realV = body
		return nil
	}
	serializer := cli.getSerializer(opts)
	return serializer.Unmarshal(ctx, httpResp, body, resp, opts)
}

func (cli *Client) handleTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	realTimeout := timeout
	if realTimeout == 0 {
		realTimeout = cli.Timeout
	}
	if realTimeout == 0 {
		return ctx, nil
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return context.WithTimeout(ctx, realTimeout)
	}
	existingTimeout := deadline.Sub(time.Now())
	if existingTimeout < realTimeout {
		return ctx, nil
	}
	return context.WithTimeout(ctx, realTimeout)
}

func (cli *Client) getHttpClient() *http.Client {
	if cli.DefaultClient != nil {
		return cli.DefaultClient
	}
	return http.DefaultClient
}

func (cli *Client) getSerializer(opts *Options) Serializer {
	if opts.Serializer != nil {
		return opts.Serializer
	}
	if cli.Serializer != nil {
		return cli.Serializer
	}
	return DefaultSerializer
}

type Interceptor func(ctx context.Context, url string, req, resp any, opts *Options, next Next) error

type Next struct {
	i            int
	interceptors []Interceptor
	handler      func(ctx context.Context, url string, req, resp any, opts *Options) error
}

func (n Next) Next(ctx context.Context, url string, req, resp any, opts *Options) error {
	if n.i >= len(n.interceptors) {
		return n.handler(ctx, url, req, resp, opts)
	}
	newN := n
	newN.i++
	return n.interceptors[n.i](ctx, url, req, resp, opts, newN)
}

type Options struct {
	Method     string
	Header     http.Header
	Timeout    time.Duration
	Serializer Serializer
	// interceptors can use custom opts to extend ability.
	CustomOpts map[any]any

	// output fields
	StatusCode int
	RespBody   []byte
	RespHeader http.Header
}

func (opts *Options) AddCustomOpt(key any, val any) *Options {
	if opts.CustomOpts == nil {
		opts.CustomOpts = make(map[any]any)
	}
	opts.CustomOpts[key] = val
	return opts
}

func (opts *Options) GetMethod() string {
	if opts.Method != "" {
		return opts.Method
	}
	return http.MethodPost
}

type Serializer interface {
	Marshal(ctx context.Context, v any, opts *Options) (string, io.Reader, error)
	Unmarshal(ctx context.Context, httpResp *http.Response, body []byte, resp any, opts *Options) error
}

type JsonSerializer struct {
}

func (s *JsonSerializer) Marshal(ctx context.Context, req any, opts *Options) (string, io.Reader, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return "", nil, err
	}
	return "application/json; charset=utf-8", bytes.NewReader(buf), nil
}

func (s *JsonSerializer) Unmarshal(ctx context.Context, httpResp *http.Response, body []byte, resp any, opts *Options) error {
	err := json.Unmarshal(body, resp)
	if err != nil {
		return fmt.Errorf("unmarshal error: %s. body is %s", err.Error(), string(body))
	}
	return nil
}
