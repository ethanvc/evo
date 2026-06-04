package evogin

import (
	"net/http"

	"github.com/ethanvc/evo/httpcli"
	"github.com/ethanvc/evo/obs"
	"github.com/gin-gonic/gin"
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
		w = newWriter(c.Writer)
		c.Writer = w
	}
	r = newReader(c.Request.Body)
	c.Request.Body = r
	defer func() {
		var err *obs.Error
		if r := recover(); r != nil {
			err = obs.New(obs.Internal, "").AppendKvEvent("Panic", obs.GetPanicPosition(0))
		}
		p.endHandle(span, c, r, w, err)
	}()
	c.Next()
}

func (p *Plugin) endHandle(span *obs.Span, c *gin.Context, r *Reader, w *Writer, err error) {
	req := obs.PreferJSON(r.Bytes())
	var respBody any
	if w != nil {
		respBody = obs.PreferJSON(w.Bytes())
	} else {
		respBody = "<ignored>"
	}
	span.SetAttr(httpcli.AttrKeyHttpMethod, c.Request.Method)
	span.SetAttr(httpcli.AttrKeyHttpUrl, c.Request.URL.String())
	span.SetAttr(httpcli.AttrKeyHttpHeader, c.Request.Header)
	span.SetAttr(httpcli.AttrKeyHttpStatusCode, c.Writer.Status())
	span.SetAttr(httpcli.AttrKeyHttpRespHeader, c.Writer.Header())
	if err == nil {
		err = p.getErrWrapper(c, w)
	}
	obs.GetObsContext(c.Request.Context()).AccessLogReport(c.Request.Context(), err, req, respBody, nil)
}

func (p *Plugin) getSpanConfigWrapper(c *gin.Context) *obs.SpanConfig {
	if p.getSpanConfig != nil {
		return p.getSpanConfig(c)
	}
	conf := &obs.SpanConfig{
		Method: c.Request.Method + " " + c.FullPath(),
	}
	return conf
}

func (p *Plugin) getErrWrapper(c *gin.Context, w *Writer) *obs.Error {
	if p.getErr != nil {
		return p.getErr(c, w)
	}
	status := c.Writer.Status()
	if status == 0 {
		return obs.New(obs.Internal, "StatusMustNotZero")
	} else if status >= http.StatusOK && status < http.StatusBadRequest {
		obs.ReportInfo(c.Request.Context(), obs.MakeKvEventStr("StatusCode", status))
		return nil
	} else if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
		return obs.New(obs.FailedPrecondition, "").AppendKvEvent("StatusCode", status)
	}
	return obs.New(obs.Internal, "").AppendKvEvent("StatusCode", status)
}

type PluginConfig struct {
	GetName       GetNameFuncT
	GetErr        func(c *gin.Context, w *Writer) (err *obs.Error)
	GetSpanConfig func(c *gin.Context) *obs.SpanConfig
}

type GetNameFuncT func(c *gin.Context) string
