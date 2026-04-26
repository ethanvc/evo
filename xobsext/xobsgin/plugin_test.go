package xobsgin

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethanvc/evo/xobs"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetErrWrapper_StatusZero(t *testing.T) {
	p := NewPlugin(&PluginConfig{})
	c, w := makeTestContextAndWriter(0)

	err := p.getErrWrapper(c, w)

	require.NotNil(t, err)
	assert.Equal(t, codes.Internal, err.GetCode())
}

func TestGetErrWrapper_Status2xx(t *testing.T) {
	statuses := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
	}
	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			p := NewPlugin(&PluginConfig{})
			c, w := makeTestContextAndWriter(status)

			err := p.getErrWrapper(c, w)

			assert.Nil(t, err, "2xx should report OK (nil error)")
		})
	}
}

func TestGetErrWrapper_Status3xx(t *testing.T) {
	statuses := []int{
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusNotModified,
	}
	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			p := NewPlugin(&PluginConfig{})
			c, w := makeTestContextAndWriter(status)

			err := p.getErrWrapper(c, w)

			assert.Nil(t, err, "3xx should report OK (nil error)")
		})
	}
}

func TestGetErrWrapper_Status4xx(t *testing.T) {
	statuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusTooManyRequests,
	}
	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			p := NewPlugin(&PluginConfig{})
			c, w := makeTestContextAndWriter(status)

			err := p.getErrWrapper(c, w)

			require.NotNil(t, err, "4xx should report FailedPrecondition")
			assert.Equal(t, codes.FailedPrecondition, err.GetCode())
		})
	}
}

func TestGetErrWrapper_Status5xx(t *testing.T) {
	statuses := []int{
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}
	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			p := NewPlugin(&PluginConfig{})
			c, w := makeTestContextAndWriter(status)

			err := p.getErrWrapper(c, w)

			require.NotNil(t, err, "5xx should report Internal")
			assert.Equal(t, codes.Internal, err.GetCode())
		})
	}
}

func TestGetErrWrapper_CustomGetErr(t *testing.T) {
	customErr := xobs.New(codes.Unavailable, "CustomErr")
	p := NewPlugin(&PluginConfig{
		GetErr: func(c *gin.Context, w *Writer) *xobs.Error {
			return customErr
		},
	})
	c, w := makeTestContextAndWriter(500)

	err := p.getErrWrapper(c, w)

	assert.Equal(t, customErr, err)
}

func TestHandle_2xx(t *testing.T) {
	rec := serveTestRequest(nil,
		func(c *gin.Context) { c.String(http.StatusOK, "ok") },
		"GET", "/test", nil,
	)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestHandle_201(t *testing.T) {
	rec := serveTestRequest(nil,
		func(c *gin.Context) { c.String(http.StatusCreated, "created") },
		"GET", "/test", nil,
	)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "created", rec.Body.String())
}

func TestHandle_3xx(t *testing.T) {
	rec := serveTestRequest(nil,
		func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/other") },
		"GET", "/test", nil,
	)
	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
}

func TestHandle_4xx(t *testing.T) {
	rec := serveTestRequest(nil,
		func(c *gin.Context) { c.String(http.StatusBadRequest, "bad request") },
		"GET", "/test", nil,
	)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "bad request", rec.Body.String())
}

func TestHandle_404(t *testing.T) {
	rec := serveTestRequest(nil,
		func(c *gin.Context) { c.String(http.StatusNotFound, "not found") },
		"GET", "/test", nil,
	)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "not found", rec.Body.String())
}

func TestHandle_5xx(t *testing.T) {
	rec := serveTestRequest(nil,
		func(c *gin.Context) { c.String(http.StatusInternalServerError, "server error") },
		"GET", "/test", nil,
	)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "server error", rec.Body.String())
}

func TestHandle_502(t *testing.T) {
	rec := serveTestRequest(nil,
		func(c *gin.Context) { c.String(http.StatusBadGateway, "bad gateway") },
		"GET", "/test", nil,
	)
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Equal(t, "bad gateway", rec.Body.String())
}

func TestHandle_ReportedErrMatchesStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantNil    bool
		wantCode   codes.Code
	}{
		{"200_nil", http.StatusOK, "ok", true, codes.OK},
		{"301_nil", http.StatusMovedPermanently, "", true, codes.OK},
		{"400_failed_precondition", http.StatusBadRequest, "bad", false, codes.FailedPrecondition},
		{"404_failed_precondition", http.StatusNotFound, "not found", false, codes.FailedPrecondition},
		{"500_internal", http.StatusInternalServerError, "err", false, codes.Internal},
		{"502_internal", http.StatusBadGateway, "gw", false, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedErr *xobs.Error
			conf := &PluginConfig{
				GetErr: func(c *gin.Context, w *Writer) *xobs.Error {
					capturedErr = defaultGetErr(c, w)
					return capturedErr
				},
			}

			handler := func(c *gin.Context) {
				if tt.status == http.StatusMovedPermanently {
					c.Redirect(tt.status, "/other")
				} else {
					c.String(tt.status, tt.body)
				}
			}
			serveTestRequest(conf, handler, "GET", "/test", nil)

			if tt.wantNil {
				assert.Nil(t, capturedErr, "expected nil error for status %d", tt.status)
			} else {
				require.NotNil(t, capturedErr, "expected non-nil error for status %d", tt.status)
				assert.Equal(t, tt.wantCode, capturedErr.GetCode())
			}
		})
	}
}

func TestHandle_RequestBodyCapture(t *testing.T) {
	body := "request body content"
	rec := serveTestRequest(nil,
		func(c *gin.Context) {
			data, _ := io.ReadAll(c.Request.Body)
			c.String(http.StatusOK, string(data))
		},
		"POST", "/test", strings.NewReader(body),
	)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, body, rec.Body.String())
}

func TestHandle_PanicRecovery(t *testing.T) {
	assert.NotPanics(t, func() {
		serveTestRequest(nil,
			func(c *gin.Context) { panic("test panic") },
			"GET", "/test", nil,
		)
	})
}

func defaultGetErr(c *gin.Context, w *Writer) *xobs.Error {
	status := w.Status()
	if status == 0 {
		return xobs.New(codes.Internal, "StatusMustNotZero")
	} else if status >= http.StatusOK && status < http.StatusBadRequest {
		return nil
	} else if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
		return xobs.New(codes.FailedPrecondition, "").AppendKvEvent("StatusCode", status)
	}
	return xobs.New(codes.Internal, "").AppendKvEvent("StatusCode", status)
}

func makeTestContextAndWriter(statusCode int) (*gin.Context, *Writer) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	w := newWriter(c.Writer)
	w.statusCode = statusCode
	return c, w
}

func serveTestRequest(conf *PluginConfig, handler gin.HandlerFunc, method, path string, body io.Reader) *httptest.ResponseRecorder {
	if conf == nil {
		conf = &PluginConfig{}
	}
	r := gin.New()
	p := NewPlugin(conf)
	r.Use(p.Handle)
	switch method {
	case "POST":
		r.POST(path, handler)
	default:
		r.GET(path, handler)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	r.ServeHTTP(rec, req)
	return rec
}
