package xobsgin

import (
	"github.com/ethanvc/evo/xobs"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
)

type Plugin struct {
	getName       GetNameFuncT
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
		err, req, resp, extra := p.getLogContentWrapper(c, w)
		xobs.GetObsContext(ctx).AccessLogReport(err, req, resp, nil, extra...)
	}()
	c.Next()
}

func (p *Plugin) getNameWrapper(c *gin.Context) string {
	if p.getName != nil {
		return p.getName(c)
	}
	return c.FullPath()
}

func toAnySlice[T any](arr []T) []any {
	res := make([]any, len(arr))
	for i, v := range arr {
		res[i] = v
	}
	return res
}

func (p *Plugin) getLogContentWrapper(c *gin.Context, w *Writer) (err error, req any, resp any, labels []xobs.KV, extra []any) {
	if p.getLogContent != nil {
		err, req, resp, labels, extra = p.getLogContent(c)
		return
	}
	if c.Writer.Status() >= 500 {
		realErr := xobs.New(codes.Internal, "")
	}
	return nil, nil, nil, nil
}

type PluginConfig struct {
	GetName       GetNameFuncT
	GetLogContent GetLogContentFuncT
}

type GetNameFuncT func(c *gin.Context) string
type GetLogContentFuncT func(c *gin.Context) (err error, req, resp any, labels []xobs.KV, extras []any)
