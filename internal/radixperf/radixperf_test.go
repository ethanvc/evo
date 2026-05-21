package radixperf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethanvc/evo/httpmux"
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
		mux.Lookup(method, path)
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
}

// ---------------------------------------------------------------------------
// benchmarks
// ---------------------------------------------------------------------------

func BenchmarkStaticRoute(b *testing.B) {
	benchAll(b, apiRoutes, "GET", "/api/v1/users")
}

func BenchmarkParamRoute(b *testing.B) {
	benchAll(b, apiRoutes, "GET", "/api/v1/users/12345")
}

func BenchmarkParamNestedRoute(b *testing.B) {
	benchAll(b, apiRoutes, "GET", "/api/v1/users/12345/orders")
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
	benchAll(b, routes, "GET", "/api/v1/resource50/12345")
}

func BenchmarkManyRoutes_Last(b *testing.B) {
	routes := manyRoutes(100)
	benchAll(b, routes, "GET", "/api/v1/resource99/12345")
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
			mux.Lookup(method, path)
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
}

func BenchmarkParallel_ParamRoute(b *testing.B) {
	b.Run("ServeMux", func(b *testing.B) {
		benchmarkRouterParallel(b, setupServeMux(apiRoutes), "GET", "/api/v1/users/12345")
	})
	b.Run("HTTPRouter", func(b *testing.B) {
		benchmarkRouterParallel(b, setupHTTPRouter(apiRoutes), "GET", "/api/v1/users/12345")
	})
	b.Run("HttpMux", func(b *testing.B) {
		benchmarkHttpMuxParallel(b, setupHttpMux(apiRoutes), "GET", "/api/v1/users/12345")
	})
	b.Run("Gin", func(b *testing.B) {
		benchmarkRouterParallel(b, setupGin(apiRoutes), "GET", "/api/v1/users/12345")
	})
}
