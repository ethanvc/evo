package evohttp

import (
	"context"
	"net/http"
)

type Server struct {
	*RouterBuilder
	noRouteHandlers HandlerChain
}

func NewServer() *Server {
	svr := &Server{}
	svr.RouterBuilder = NewRouterBuilder()
	return svr
}

func (svr *Server) Use(handlers ...Handler) {
	svr.RouterBuilder.Use(handlers...)
	svr.noRouteHandlers = append(svr.noRouteHandlers, handlers...)
}

func (svr *Server) UseF(handlers ...HandlerFunc) {
	svr.Use(funcToHandlers(handlers)...)
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
	info.Writer.Reset(w)
	c := context.WithValue(req.Context(), contextKeyRequestInfo{}, info)
	n := svr.router.Find(req.Method, req.URL.Path, info.params)
	if n == nil {
		svr.serveHandlerNotFound(c, info)
		return
	}
	info.handlers = n.handlers
	info.PatternPath = n.fullPath
	info.Next(c, info)
	if info.Writer.GetStatus() == 0 {
		info.Writer.WriteHeader(http.StatusInternalServerError)
	}
}

func (svr *Server) serveHandlerNotFound(c context.Context, info *RequestInfo) {
	info.handlers = svr.noRouteHandlers
	info.Next(c, info)
	if info.Writer.GetStatus() == 0 {
		info.Writer.WriteHeader(http.StatusNotFound)
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
