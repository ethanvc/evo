package httpcli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/plog"
	"google.golang.org/grpc/codes"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type SingleAttempt struct {
	Request  *http.Request
	Response *http.Response
	err      error
	RespBody []byte
	Template *HttpTemplate
	Conf     *Config
}

func NewSingleAttempt(c context.Context, httpMethod, url string, conf *Config) *SingleAttempt {
	sa := &SingleAttempt{
		Template: DefaultTemplate,
		Conf:     conf,
	}
	c = plog.WithLogContext(c, &plog.LogContextConfig{
		Method: conf.GetPatternUrl(url),
	})
	// make error processing easier
	sa.Request, sa.err = http.NewRequestWithContext(c, httpMethod, url, nil)
	if sa.err != nil {
		sa.Request, _ = http.NewRequestWithContext(c, httpMethod, "http://127.0.0.1:1/badurl_or_method_offered", nil)
	}
	return sa
}

func (sa *SingleAttempt) GetResponseCode() int {
	if sa.Response != nil {
		return sa.Response.StatusCode
	} else {
		return 0
	}
}

func (sa *SingleAttempt) Do(req any, resp any) (err error) {
	return sa.Template.Do(req, resp, sa)
}

func Do[Resp any](sa *SingleAttempt, req any) (Resp, error) {
	var resp Resp
	err := sa.Do(req, &resp)
	return resp, err
}

var DefaultTemplate *HttpTemplate = NewHttpTemplate()

type HttpTemplate struct {
	Client  *http.Client
	Encoder func(req any, sa *SingleAttempt) error
	Decoder func(sa *SingleAttempt) error
	Report  func(err error, sa *SingleAttempt, req, resp any)
}

func NewHttpTemplate() *HttpTemplate {
	return &HttpTemplate{
		Client: http.DefaultClient,
	}
}

func (template *HttpTemplate) Do(req, resp any, sa *SingleAttempt) (err error) {
	err = template.realDo(req, resp, sa)
	template.report(err, sa, req, resp)
	return err
}

func (template *HttpTemplate) report(err error, sa *SingleAttempt, req, resp any) {
	if template.Report != nil {
		template.Report(err, sa, req, resp)
		return
	}
	plog.DefaultRequestLogger().Log(sa.Request.Context(), &plog.RequestLogInfo{
		Err:  err,
		Req:  req,
		Resp: resp,
	}, slog.String("code", http.StatusText(sa.GetResponseCode())))
}

func (template *HttpTemplate) realDo(req, resp any, sa *SingleAttempt) (err error) {
	if sa.err != nil {
		return sa.err
	}
	err = template.encoder(req, sa)
	if err != nil {
		return err
	}
	sa.Response, err = template.Client.Do(sa.Request)
	return template.decoder(resp, sa)
}

func (template *HttpTemplate) encoder(req any, sa *SingleAttempt) error {
	if template.Encoder != nil {
		return template.Encoder(req, sa)
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

func (template *HttpTemplate) decoder(resp any, sa *SingleAttempt) error {
	if resp == nil {
		return nil
	}
	if template.Decoder != nil {
		return template.Decoder(sa)
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

type Config struct {
	patternUrl string
}

func NewConfig() *Config {
	return &Config{}
}

func (conf *Config) SetPatternUrl(url string) *Config {
	conf.patternUrl = url
	return conf
}

func (conf *Config) GetPatternUrl(url string) string {
	if conf != nil && conf.patternUrl != "" {
		return conf.patternUrl
	} else {
		return url
	}
}
