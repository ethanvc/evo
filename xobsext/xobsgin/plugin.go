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
	getLogContent GetLogContentFuncT
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
}

func (p *Plugin) Handle(c *gin.Context) {
	ctx := xobs.WithSpanContext(c.Request.Context(), &xobs.SpanConfig{
		Name: p.getNameWrapper(c),
	})
	c.Request = c.Request.WithContext(ctx)
	w := newWriter(c.Writer)
	c.Writer = w
	defer func() {
		req, resp, labels, extra := p.getLogContentWrapper(c, w)
		err := p.getErrWrapper(c, w)
		xobs.GetObsContext(ctx).AccessLogReport(err, req, resp, labels, extra...)
	}()
	c.Next()
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

func (p *Plugin) getLogContentWrapper(c *gin.Context, w *Writer) (req any, resp any, labels []xobs.KV, extra []any) {
	if p.getLogContent != nil {
		req, resp, labels, extra = p.getLogContent(c)
		return
	}
	return nil, nil, nil, nil
}

type PluginConfig struct {
	GetName       GetNameFuncT
	GetErr        func(c *gin.Context, w *Writer) (err *xobs.Error)
	GetLogContent GetLogContentFuncT
}

type GetNameFuncT func(c *gin.Context) string
type GetLogContentFuncT func(c *gin.Context) (req, resp any, labels []xobs.KV, extras []any)
