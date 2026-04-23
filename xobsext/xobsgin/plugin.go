package xobsgin

import (
	"github.com/ethanvc/evo/xobs"
	"github.com/gin-gonic/gin"
)

type Plugin struct {
	getName GetNameFuncT
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
	defer func() {
		xobs.GetObsContext(ctx).LogReportAccessLog()
	}()
	c.Next()
}

func (p *Plugin) getNameWrapper(c *gin.Context) string {
	if p.getName != nil {
		return p.getName(c)
	}
	return c.FullPath()
}

type PluginConfig struct {
	GetName GetNameFuncT
}

type GetNameFuncT func(c *gin.Context) string
type GetResultFuncT func(c *gin.Context) (err error, req, resp any)
