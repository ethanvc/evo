package evogin

import (
	"net/http"

	"github.com/ethanvc/evo/obs"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
)

type Plugin struct {
	getName       GetNameFuncT
	getErr        func(c *gin.Context, w *Writer) *obs.Error
	getSpanConfig func(c *gin.Context) *obs.SpanConfig
}

func NewPlugin(conf *PluginConfig) *Plugin {
	p := &Plugin{}
	p.init(conf)
	return p
}

func (p *Plugin) init(conf *PluginConfig) {
	p.getName = conf.GetName
	if p.getName == nil {
		p.getName = func(c *gin.Context) string {
			return c.FullPath()
		}
	}
	p.getErr = conf.GetErr
	p.getSpanConfig = conf.GetSpanConfig
}

func (p *Plugin) Handle(c *gin.Context) {
	patternConfig := getPatternConfig(c.Request.Method, c.FullPath())
	spanConfig := p.getSpanConfigWrapper(c)
	ctx, span := obs.WithSpan(c.Request.Context(), spanConfig)
	c.Request = c.Request.WithContext(ctx)
	var w *Writer
	var r *Reader
	if !patternConfig.GetIgnoreResponseLog() {
		w := newWriter(c.Writer)
		c.Writer = w
	}
	r = newReader(c.Request.Body)
	c.Request.Body = r
	defer func() {
		var err *obs.Error
		if r := recover(); r != nil {
			err = obs.New(codes.Internal, "").AppendKvEvent("Panic", obs.GetPanicPosition(0))
		}
		p.endHandle(span, c, r, w, err)
	}()
	c.Next()
}

func (p *Plugin) endHandle(span *obs.Span, c *gin.Context, r *Reader, w *Writer, err error) {
	req := r.Bytes()
	var resp []byte
	if w != nil {
		resp = w.Bytes()
	} else {
		resp = []byte("<ignored>")
	}
	var extra []any
	extra = append(extra, "http_url", c.Request.URL.String())
	extra = append(extra, "http_status_code", w.Status())
	extra = append(extra, "http_req_header", c.Request.Header)
	extra = append(extra, "http_resp_header", w.Header())
	extra = append(extra, "client_ip", c.Request.RemoteAddr)
	if err == nil {
		err = p.getErrWrapper(c, w)
	}
	obs.GetObsContext(c.Request.Context()).AccessLogReport(c.Request.Context(), err, req, resp, nil, extra...)
}

func (p *Plugin) getSpanConfigWrapper(c *gin.Context) *obs.SpanConfig {
	if p.getSpanConfig != nil {
		return p.getSpanConfig(c)
	}
	conf := &obs.SpanConfig{
		Method: c.FullPath(),
	}
	return conf
}

func (p *Plugin) getErrWrapper(c *gin.Context, w *Writer) *obs.Error {
	if p.getErr != nil {
		return p.getErr(c, w)
	}
	status := w.Status()
	if status == 0 {
		return obs.New(codes.Internal, "StatusMustNotZero")
	} else if status >= http.StatusOK && status < http.StatusBadRequest {
		obs.ReportInfo(c.Request.Context(), obs.MakeKvEventStr("StatusCode", status))
		return nil
	} else if status >= http.StatusBadRequest && w.Status() < http.StatusInternalServerError {
		return obs.New(codes.FailedPrecondition, "").AppendKvEvent("StatusCode", status)
	}
	return obs.New(codes.Internal, "").AppendKvEvent("StatusCode", status)
}

type PluginConfig struct {
	GetName       GetNameFuncT
	GetErr        func(c *gin.Context, w *Writer) (err *obs.Error)
	GetSpanConfig func(c *gin.Context) *obs.SpanConfig
}

type GetNameFuncT func(c *gin.Context) string
