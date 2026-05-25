package radixperf

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethanvc/evo/httpmux"
	"github.com/ethanvc/evo/patternmux"
	"github.com/gin-gonic/gin"
	"github.com/julienschmidt/httprouter"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// ---------------------------------------------------------------------------
// route definitions (neutral format, converted per-router)
// ---------------------------------------------------------------------------

type routeDef struct {
	method string
	path   string // uses :param style; converted to {param} for ServeMux
}

var apiRoutes = []routeDef{
	{"GET", "/"},
	{"GET", "/health"},
	{"GET", "/metrics"},
	{"GET", "/api/v1/users"},
	{"POST", "/api/v1/users"},
	{"GET", "/api/v1/users/:id"},
	{"PUT", "/api/v1/users/:id"},
	{"DELETE", "/api/v1/users/:id"},
	{"GET", "/api/v1/users/:id/profile"},
	{"GET", "/api/v1/users/:id/orders"},
	{"GET", "/api/v1/orders"},
	{"POST", "/api/v1/orders"},
	{"GET", "/api/v1/orders/:id"},
	{"GET", "/api/v1/products"},
	{"GET", "/api/v1/products/:id"},
	{"GET", "/api/v1/products/:id/reviews"},
}

func manyRoutes(n int) []routeDef {
	routes := make([]routeDef, 0, n*3)
	for i := range n {
		res := fmt.Sprintf("resource%d", i)
		routes = append(routes,
			routeDef{"GET", "/api/v1/" + res},
			routeDef{"POST", "/api/v1/" + res},
			routeDef{"GET", "/api/v1/" + res + "/:id"},
		)
	}
	return routes
}

// ---------------------------------------------------------------------------
// router setup helpers
// ---------------------------------------------------------------------------

func toServeMuxPattern(method, path string) string {
	out := make([]byte, 0, len(method)+1+len(path)+8)
	out = append(out, method...)
	out = append(out, ' ')
	for i := 0; i < len(path); i++ {
		if path[i] == ':' {
			j := i + 1
			for j < len(path) && path[j] != '/' {
				j++
			}
			out = append(out, '{')
			out = append(out, path[i+1:j]...)
			out = append(out, '}')
			i = j - 1
		} else {
			out = append(out, path[i])
		}
	}
	return string(out)
}

func setupServeMux(routes []routeDef) http.Handler {
	mux := http.NewServeMux()
	for _, r := range routes {
		mux.HandleFunc(toServeMuxPattern(r.method, r.path),
			func(w http.ResponseWriter, r *http.Request) {})
	}
	return mux
}

func setupHTTPRouter(routes []routeDef) http.Handler {
	router := httprouter.New()
	nop := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {}
	for _, r := range routes {
		router.Handle(r.method, r.path, nop)
	}
	return router
}

func setupGin(routes []routeDef) http.Handler {
	r := gin.New()
	nop := func(c *gin.Context) {}
	for _, rd := range routes {
		r.Handle(rd.method, rd.path, nop)
	}
	return r
}

func setupHttpMux(routes []routeDef) *httpmux.HttpMux[int] {
	mux := httpmux.New[int]()
	for i, r := range routes {
		mux.Handle(r.method, r.path, i+1)
	}
	return mux
}

// toPatternMuxPattern converts :param / *catch-all paths to patternmux syntax.
func toPatternMuxPattern(path string) string {
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case ':':
			j := i + 1
			for j < len(path) && path[j] != '/' {
				j++
			}
			b.WriteString("{replace::")
			b.WriteString(path[i+1 : j])
			b.WriteString(";until-slash}")
			i = j - 1
		case '*':
			j := i + 1
			for j < len(path) && path[j] != '/' {
				j++
			}
			b.WriteString("{replace:*")
			b.WriteString(path[i+1 : j])
			b.WriteString(";rest}")
			i = j - 1
		default:
			b.WriteByte(path[i])
		}
	}
	return b.String()
}

// setupPatternMux registers path-only patterns. Duplicate paths (e.g. GET+POST same
// path) are skipped because patternmux has no HTTP method dimension.
func setupPatternMux(routes []routeDef) *patternmux.Mux[int] {
	mux := patternmux.New[int]()
	for i, r := range routes {
		if err := mux.Register(toPatternMuxPattern(r.path), i+1); err != nil {
			if errors.Is(err, patternmux.ErrDuplicatePattern) {
				continue
			}
			panic(err)
		}
	}
	return mux
}

// httpMuxBenchHandler adapts HttpMux to http.Handler for benchmarks.
type httpMuxBenchHandler struct {
	mux *httpmux.HttpMux[int]
}

func newHttpMuxBenchHandler(routes []routeDef) *httpMuxBenchHandler {
	return &httpMuxBenchHandler{mux: setupHttpMux(routes)}
}

func (h *httpMuxBenchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, params, _ := h.mux.Lookup(r.Method, r.URL.Path)
	if params == nil {
		return
	}
	_ = (*params)[0].Value
	httpmux.PutParams(params)
}

func setupHttpMuxHandler(routes []routeDef) http.Handler {
	return newHttpMuxBenchHandler(routes)
}

// patternMuxBenchHandler adapts patternmux to http.Handler for param-route benchmarks.
type patternMuxBenchHandler struct {
	mux *patternmux.Mux[int]
}

func newPatternMuxBenchHandler(routes []routeDef) *patternMuxBenchHandler {
	return &patternMuxBenchHandler{mux: setupPatternMux(routes)}
}

func (h *patternMuxBenchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, caps, _, ok := h.mux.Lookup(r.URL.Path)
	if !ok {
		return
	}
	if caps != nil {
		_ = (*caps)[0].Value
		patternmux.PutCaptures(caps)
	}
}

// ---------------------------------------------------------------------------
// benchmark runner
// ---------------------------------------------------------------------------

func benchmarkRouter(b *testing.B, handler http.Handler, method, path string) {
	b.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		handler.ServeHTTP(w, req)
	}
}

func benchmarkHttpMux(b *testing.B, mux *httpmux.HttpMux[int], method, path string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, params, _ := mux.Lookup(method, path)
		if params != nil {
			httpmux.PutParams(params)
		}
	}
}

func benchmarkPatternMux(b *testing.B, mux *patternmux.Mux[int], path string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, caps, _, ok := mux.Lookup(path)
		if caps != nil {
			patternmux.PutCaptures(caps)
		}
		_ = ok
	}
}

func benchAll(b *testing.B, routes []routeDef, method, path string) {
	b.Run("ServeMux", func(b *testing.B) {
		benchmarkRouter(b, setupServeMux(routes), method, path)
	})
	b.Run("HTTPRouter", func(b *testing.B) {
		benchmarkRouter(b, setupHTTPRouter(routes), method, path)
	})
	b.Run("HttpMux", func(b *testing.B) {
		benchmarkHttpMux(b, setupHttpMux(routes), method, path)
	})
	b.Run("Gin", func(b *testing.B) {
		benchmarkRouter(b, setupGin(routes), method, path)
	})
	b.Run("PatternMux", func(b *testing.B) {
		benchmarkPatternMux(b, setupPatternMux(routes), path)
	})
}

func benchAllParam(b *testing.B, routes []routeDef, method, path string) {
	b.Run("ServeMux", func(b *testing.B) {
		benchmarkRouter(b, setupServeMux(routes), method, path)
	})
	b.Run("HTTPRouter", func(b *testing.B) {
		benchmarkRouter(b, setupHTTPRouter(routes), method, path)
	})
	b.Run("HttpMux", func(b *testing.B) {
		benchmarkRouter(b, setupHttpMuxHandler(routes), method, path)
	})
	b.Run("Gin", func(b *testing.B) {
		benchmarkRouter(b, setupGin(routes), method, path)
	})
	b.Run("PatternMux", func(b *testing.B) {
		benchmarkRouter(b, newPatternMuxBenchHandler(routes), method, path)
	})
}

// ---------------------------------------------------------------------------
// benchmarks
// ---------------------------------------------------------------------------

func BenchmarkStaticRoute(b *testing.B) {
	benchAll(b, apiRoutes, "GET", "/api/v1/users")
}

func BenchmarkParamRoute(b *testing.B) {
	benchAllParam(b, apiRoutes, "GET", "/api/v1/users/12345")
}

func BenchmarkParamNestedRoute(b *testing.B) {
	benchAllParam(b, apiRoutes, "GET", "/api/v1/users/12345/orders")
}

func BenchmarkRootRoute(b *testing.B) {
	benchAll(b, apiRoutes, "GET", "/")
}

func BenchmarkShortStaticRoute(b *testing.B) {
	benchAll(b, apiRoutes, "GET", "/health")
}

func BenchmarkManyRoutes_Static(b *testing.B) {
	routes := manyRoutes(100)
	benchAll(b, routes, "GET", "/api/v1/resource50")
}

func BenchmarkManyRoutes_Param(b *testing.B) {
	routes := manyRoutes(100)
	benchAllParam(b, routes, "GET", "/api/v1/resource50/12345")
}

func BenchmarkManyRoutes_Last(b *testing.B) {
	routes := manyRoutes(100)
	benchAllParam(b, routes, "GET", "/api/v1/resource99/12345")
}

// ---------------------------------------------------------------------------
// parallel benchmarks
// ---------------------------------------------------------------------------

func benchmarkRouterParallel(b *testing.B, handler http.Handler, method, path string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		for pb.Next() {
			handler.ServeHTTP(w, req)
		}
	})
}

func benchmarkHttpMuxParallel(b *testing.B, mux *httpmux.HttpMux[int], method, path string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, params, _ := mux.Lookup(method, path)
			if params != nil {
				httpmux.PutParams(params)
			}
		}
	})
}

func BenchmarkParallel_StaticRoute(b *testing.B) {
	b.Run("ServeMux", func(b *testing.B) {
		benchmarkRouterParallel(b, setupServeMux(apiRoutes), "GET", "/api/v1/users")
	})
	b.Run("HTTPRouter", func(b *testing.B) {
		benchmarkRouterParallel(b, setupHTTPRouter(apiRoutes), "GET", "/api/v1/users")
	})
	b.Run("HttpMux", func(b *testing.B) {
		benchmarkHttpMuxParallel(b, setupHttpMux(apiRoutes), "GET", "/api/v1/users")
	})
	b.Run("Gin", func(b *testing.B) {
		benchmarkRouterParallel(b, setupGin(apiRoutes), "GET", "/api/v1/users")
	})
	b.Run("PatternMux", func(b *testing.B) {
		benchmarkPatternMuxParallel(b, setupPatternMux(apiRoutes), "/api/v1/users")
	})
}

func benchmarkHttpMuxParamParallel(b *testing.B, routes []routeDef, method, path string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		h := newHttpMuxBenchHandler(routes)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		for pb.Next() {
			h.ServeHTTP(w, req)
		}
	})
}

func benchmarkHTTPRouterLookup(b *testing.B, router *httprouter.Router, method, path string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		router.Lookup(method, path)
	}
}

func setupHTTPRouterRaw(routes []routeDef) *httprouter.Router {
	router := httprouter.New()
	nop := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {}
	for _, r := range routes {
		router.Handle(r.method, r.path, nop)
	}
	return router
}

// BenchmarkParamRoute_Lookup compares Lookup-to-Lookup on the same tree implementation.
func BenchmarkParamRoute_Lookup(b *testing.B) {
	b.Run("HTTPRouter", func(b *testing.B) {
		benchmarkHTTPRouterLookup(b, setupHTTPRouterRaw(apiRoutes), "GET", "/api/v1/users/12345")
	})
	b.Run("HttpMux", func(b *testing.B) {
		benchmarkHttpMux(b, setupHttpMux(apiRoutes), "GET", "/api/v1/users/12345")
	})
	b.Run("PatternMux", func(b *testing.B) {
		benchmarkPatternMux(b, setupPatternMux(apiRoutes), "/api/v1/users/12345")
	})
}

func BenchmarkParallel_ParamRoute(b *testing.B) {
	b.Run("ServeMux", func(b *testing.B) {
		benchmarkRouterParallel(b, setupServeMux(apiRoutes), "GET", "/api/v1/users/12345")
	})
	b.Run("HTTPRouter", func(b *testing.B) {
		benchmarkRouterParallel(b, setupHTTPRouter(apiRoutes), "GET", "/api/v1/users/12345")
	})
	b.Run("HttpMux", func(b *testing.B) {
		benchmarkHttpMuxParamParallel(b, apiRoutes, "GET", "/api/v1/users/12345")
	})
	b.Run("Gin", func(b *testing.B) {
		benchmarkRouterParallel(b, setupGin(apiRoutes), "GET", "/api/v1/users/12345")
	})
	b.Run("PatternMux", func(b *testing.B) {
		benchmarkPatternMuxParamParallel(b, apiRoutes, "GET", "/api/v1/users/12345")
	})
}

func benchmarkPatternMuxParallel(b *testing.B, mux *patternmux.Mux[int], path string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, caps, _, ok := mux.Lookup(path)
			if caps != nil {
				patternmux.PutCaptures(caps)
			}
			_ = ok
		}
	})
}

func benchmarkPatternMuxParamParallel(b *testing.B, routes []routeDef, method, path string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		h := newPatternMuxBenchHandler(routes)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		for pb.Next() {
			h.ServeHTTP(w, req)
		}
	})
}
