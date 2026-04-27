package xobsgin

import (
	"net/http"

	"github.com/ethanvc/evo/xobs"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
)

type Plugin struct {
	getName       GetNameFuncT
	getErr        func(c *gin.Context, w *Writer) *xobs.Error
	getSpanConfig func(c *gin.Context) *xobs.SpanConfig
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
	spanConfig := p.getSpanConfigWrapper(c)
	ctx := xobs.WithSpanContext(c.Request.Context(), spanConfig)
	c.Request = c.Request.WithContext(ctx)
	w := newWriter(c.Writer)
	c.Writer = w
	r := newReader(c.Request.Body)
	c.Request.Body = r
	defer func() {
		var err *xobs.Error
		if r := recover(); r != nil {
			err = xobs.New(codes.Internal, "").AppendKvEvent("Panic", xobs.GetPanicPosition(0))
		}
		req, resp, labels, extra := p.getLogContentWrapper(c, r, w)
		if err == nil {
			err = p.getErrWrapper(c, w)
		}
		xobs.GetObsContext(ctx).AccessLogReport(ctx, err, req, resp, labels, extra...)
	}()
	c.Next()
}

func (p *Plugin) getSpanConfigWrapper(c *gin.Context) *xobs.SpanConfig {
	if p.getSpanConfig != nil {
		return p.getSpanConfig(c)
	}
	conf := &xobs.SpanConfig{
		Method: c.FullPath(),
	}
	return conf
}

func (p *Plugin) getErrWrapper(c *gin.Context, w *Writer) *xobs.Error {
	if p.getErr != nil {
		return p.getErr(c, w)
	}
	status := w.Status()
	if status == 0 {
		return xobs.New(codes.Internal, "StatusMustNotZero")
	} else if status >= http.StatusOK && status < http.StatusBadRequest {
		xobs.ReportInfo(c.Request.Context(), xobs.MakeKvEventStr("StatusCode", status))
		return nil
	} else if status >= http.StatusBadRequest && w.Status() < http.StatusInternalServerError {
		return xobs.New(codes.FailedPrecondition, "").AppendKvEvent("StatusCode", status)
	}
	return xobs.New(codes.Internal, "").AppendKvEvent("StatusCode", status)
}

func (p *Plugin) getNameWrapper(c *gin.Context) string {
	if p.getName != nil {
		return p.getName(c)
	}
	return c.FullPath()
}

func (p *Plugin) getLogContentWrapper(c *gin.Context, r *Reader, w *Writer) (req any, resp any, labels []xobs.KV, extra []any) {
	req = r.Bytes()
	resp = w.Bytes()
	extra = append(extra, "http_url", c.Request.URL.String())
	extra = append(extra, "http_status_code", w.Status())
	extra = append(extra, "http_req_header", c.Request.Header)
	extra = append(extra, "http_resp_header", w.Header())
	extra = append(extra, "client_ip", c.Request.RemoteAddr)
	return
}

type PluginConfig struct {
	GetName       GetNameFuncT
	GetErr        func(c *gin.Context, w *Writer) (err *xobs.Error)
	GetSpanConfig func(c *gin.Context) *xobs.SpanConfig
}

type GetNameFuncT func(c *gin.Context) string
