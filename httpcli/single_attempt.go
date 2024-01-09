package httpcli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ethanvc/evo/base"
	"google.golang.org/grpc/codes"
	"io"
	"net/http"
	"strings"
)

type SingleAttempt struct {
	Request  *http.Request
	Response *http.Response
	err      error
	RespBody []byte
	Template *HttpTemplate
}

func NewSingleAttempt(c context.Context, httpMethod, url string) *SingleAttempt {
	if c == nil {
		c = context.Background()
	}
	sa := &SingleAttempt{}
	sa.Template = DefaultTemplate
	var err error
	// make error processing easier
	sa.Request, err = http.NewRequestWithContext(c, httpMethod, url, nil)
	if err != nil {
		sa.err = err
		sa.Request, _ = http.NewRequestWithContext(c, httpMethod, "http://127.0.0.1:1/badurl_or_method_offered", nil)
	}
	return sa
}

func (sa *SingleAttempt) Do(req any, resp any) (err error) {
	if sa.err != nil {
		return sa.err
	}
	return sa.Template.Do(sa.Request.Context(), req, resp, sa)
}

func Do[Resp any](sa *SingleAttempt, req any) (Resp, error) {
	var resp Resp
	err := sa.Do(req, &resp)
	return resp, err
}

var DefaultTemplate *HttpTemplate = NewHttpTemplate()

type HttpTemplate struct {
	Client  *http.Client
	Encoder func(ctx context.Context, req any, sa *SingleAttempt) error
	Decoder func(ctx context.Context, sa *SingleAttempt) error
}

func NewHttpTemplate() *HttpTemplate {
	return &HttpTemplate{
		Client: http.DefaultClient,
	}
}

func (template *HttpTemplate) Do(c context.Context, req, resp any, sa *SingleAttempt) error {
	err := template.encoder(c, req, sa)
	if err != nil {
		return err
	}
	sa.Response, err = template.Client.Do(sa.Request)
	return template.decoder(c, resp, sa)
}

func (template *HttpTemplate) encoder(c context.Context, req any, sa *SingleAttempt) error {
	if template.Encoder != nil {
		return template.Encoder(c, req, sa)
	}
	if req == nil || sa.Request.Body != nil {
		return nil
	}
	switch realReq := req.(type) {
	case string:
		sa.Request.Body = io.NopCloser(strings.NewReader(realReq))
	case []byte:
		sa.Request.Body = io.NopCloser(bytes.NewReader(realReq))
	default:
		buf, err := json.Marshal(req)
		if err != nil {
			return base.New(codes.InvalidArgument, "UnsupportRequestType").SetMsg(err.Error()).Err()
		}
		sa.Request.Body = io.NopCloser(bytes.NewReader(buf))
		sa.Request.Header.Set("Content-Type", "application/json")
	}
	// goway retry support for http2
	sa.Request.GetBody = func() (io.ReadCloser, error) {
		return sa.Request.Body, nil
	}
	return nil
}

func (template *HttpTemplate) decoder(c context.Context, resp any, sa *SingleAttempt) error {
	if resp == nil {
		return nil
	}
	if template.Decoder != nil {
		return template.Decoder(c, sa)
	}
	buf, err := io.ReadAll(sa.Response.Body)
	sa.Response.Body.Close()
	sa.RespBody = buf
	if err != nil {
		return err
	}
	switch realResp := resp.(type) {
	case *string:
		*realResp = string(buf)
	case *[]byte:
		*realResp = buf
	default:
		err := json.Unmarshal(buf, resp)
		if err != nil {
			return err
		}
	}
	return nil
}
