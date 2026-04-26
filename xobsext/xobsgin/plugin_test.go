package xobsgin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func Test_Return200(t *testing.T) {
	engine := gin.New()
	engine.Use(NewPlugin(&PluginConfig{}).Handle)
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})
	writer := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(writer, httpReq)
}
