package evohttp

import (
	"context"
	"net/http"
	"time"

	"github.com/ethanvc/evo/evolog"
)

type Server struct {
	*RouterBuilder
	noRouteHandlers     HandlerChain
	userNoRouteHandlers HandlerChain
}

func NewServer() *Server {
	svr := &Server{
		noRouteHandlers: []Handler{HandlerFunc(finalNoRouteHandler)},
	}
	svr.RouterBuilder = NewRouterBuilder()
	svr.Use(NewLogHandler(), NewCodecHandler(), NewRecoverHandler())
	return svr
}

func (svr *Server) Use(handlers ...Handler) {
	svr.RouterBuilder.Use(handlers...)
	svr.rebuild404Handlers()
}

func (svr *Server) UseF(handlers ...HandlerFunc) {
	svr.Use(funcToHandlers(handlers)...)
}

func (svr *Server) NoRoute(handler ...Handler) {
	svr.userNoRouteHandlers = append(svr.userNoRouteHandlers, handler...)
	svr.rebuild404Handlers()
}

func (svr *Server) Run(addr string) {
	err := http.ListenAndServe(addr, http.HandlerFunc(svr.ServeHTTP))
	if err != nil {
		panic(err)
	}
}

func (svr *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	info := NewRequestInfo()
	info.Request = req
	info.RequestTime = time.Now()
	info.Writer.Reset(w)
	c := context.WithValue(req.Context(), contextKeyRequestInfo{}, info)
	c = evolog.WithLogContext(c, &evolog.LogContextConfig{
		TraceId: req.Header.Get("x-trace-id"),
	})
	svr.route(c, info)
}

func (svr *Server) route(c context.Context, info *RequestInfo) {
	n := svr.router.Find(info.Request.Method, info.Request.URL.Path, info.params)
	if n == nil {
		info.ResetHandlers(svr.noRouteHandlers)
		info.Next(c, nil)
		return
	}
	info.ResetHandlers(n.handlers)
	info.PatternPath = n.fullPath
	evolog.GetLogContext(c).SetMethod(n.fullPath)
	info.Next(c, nil)
}

func (svr *Server) rebuild404Handlers() {
	if len(svr.userNoRouteHandlers) == 0 {
		svr.noRouteHandlers = append([]Handler{HandlerFunc(finalNoRouteHandler)}, svr.RouterBuilder.handlers...)
	} else {
		svr.noRouteHandlers = joinSlice(svr.RouterBuilder.handlers, svr.userNoRouteHandlers)
	}
}

type Handler interface {
	HandleRequest(c context.Context, req any, info *RequestInfo) (any, error)
}

type HandlerChain []Handler

type HandlerFunc func(context.Context, any, *RequestInfo) (any, error)

func (h HandlerFunc) HandleRequest(c context.Context, req any, info *RequestInfo) (any, error) {
	return h(c, req, info)
}

func finalNoRouteHandler(c context.Context, req any, info *RequestInfo) (resp any, err error) {
	resp, err = info.Next(c, req)
	if info.Writer.GetStatus() == 0 {
		info.Writer.WriteHeader(http.StatusNotFound)
	}
	return
}
