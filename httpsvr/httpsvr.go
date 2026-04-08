package httpsvr

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/ethanvc/study/golangproj/httpsvr/ginradix"
)

type Server struct {
	Serializer      Serializer
	Logger          Logger
	Timeout         time.Duration
	Interceptors    []Interceptor
	NotFoundHandler *Handler
	router          Router
}

func (s *Server) Register(pattern string, f any, methodSlice ...string) {
	h := NewHandler(f)
	s.router.Register(pattern, h, methodSlice...)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, pattern, params := s.router.Get(r.Method, r.URL.Path)
	if h == nil {
		h, pattern, params = s.getNotFoundHandler()
	}
	callInfo := &CallInfo{
		Pattern:   pattern,
		Request:   r,
		Writer:    w,
		Server:    s,
		PathParms: params,
	}
	callInfo.RespHeader = w.Header()
	ctx := context.WithValue(r.Context(), contextKeyCallInfo{}, callInfo)
	_ = h.Handle(ctx, callInfo)
}

func (s *Server) getNotFoundHandler() (*Handler, string, ginradix.Params) {
	h := DefaultNotFoundHandler
	if s.NotFoundHandler != nil {
		h = s.NotFoundHandler
	}
	return h, "/UnknownPath", nil
}

func (s *Server) getLogger() Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return DefaultLogger
}

func (s *Server) getSerializer() Serializer {
	if s.Serializer != nil {
		return s.Serializer
	}
	return DefaultSerializer
}

type Interceptor func(ctx context.Context, req any, info *CallInfo, next Next) (any, error)

type Next struct {
	i            int
	interceptors []Interceptor
	handler      *Handler
}

func (n Next) Next(ctx context.Context, req any, info *CallInfo) (any, error) {
	if n.i >= len(n.interceptors) {
		return n.handler.call(ctx, req)
	}
	newN := n
	newN.i++
	return n.interceptors[n.i](ctx, req, info, newN)
}

func validateAndParseFunc(f any) (reqType reflect.Type, err error) {
	funcType := reflect.TypeOf(f)
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("the input must be a function type: %T", f)
	}

	if funcType.NumIn() != 2 {
		return nil, fmt.Errorf("the function must have exactly two parameter")
	}

	firstParam := funcType.In(0)
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()

	if firstParam != contextType {
		return nil, fmt.Errorf("the first parameter must be context.Context")
	}

	secondParam := funcType.In(1)
	if secondParam.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("the second parameter must be a pointer")
	}

	if funcType.NumOut() != 2 {
		return nil, fmt.Errorf("function must return exact two params")
	}

	firstReturn := funcType.Out(0)
	if firstReturn.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("the first return param must be a pointer")
	}

	secondReturn := funcType.Out(1)
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if secondReturn != errorType {
		return nil, fmt.Errorf("the second return param must be error type")
	}

	return secondParam, nil
}

type CallInfo struct {
	Pattern     string
	Request     *http.Request
	Writer      http.ResponseWriter
	PathParms   ginradix.Params
	Server      *Server
	RequestBody []byte

	StatusCode   int
	RespHeader   http.Header
	ResponseBody []byte
}

type contextKeyCallInfo struct{}

func GetCallInfo(ctx context.Context) *CallInfo {
	info, _ := ctx.Value(contextKeyCallInfo{}).(*CallInfo)
	return info
}

type Serializer interface {
	Marshal(ctx context.Context, err error, v any, info *CallInfo) (io.ReadCloser, error)
	GetStatusCode(ctx context.Context, err error) int
	Unmarshal(ctx context.Context, v any, info *CallInfo) error
}

type Logger interface {
	Start(ctx context.Context, info *CallInfo) context.Context
	End(ctx context.Context, err error, req, resp any, info *CallInfo)
	Log(ctx context.Context, lvl slog.Level, event string, args ...any)
}
