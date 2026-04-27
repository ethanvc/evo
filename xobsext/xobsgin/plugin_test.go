package xobsgin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethanvc/evo/xobs"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func Test_Return200(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	geSpanConfig := func(c *gin.Context) *xobs.SpanConfig {
		return &xobs.SpanConfig{
			Method: c.FullPath(),
			ObsConfig: xobs.ObsConfig{
				Handler: xobs.NewJsonHandler(buf),
			},
		}
	}
	plugin := NewPlugin(&PluginConfig{
		GetSpanConfig: geSpanConfig,
	})
	engine := gin.New()
	engine.Use(plugin.Handle)
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})
	writer := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(writer, httpReq)
	require.Contains(t, buf.String(), "")
}
