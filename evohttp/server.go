package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
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
	var err error
	defer func() {
		if err == nil || info.Writer.GetStatus() != 0 {
			return
		}
		info.Writer.WriteHeader(http.StatusInternalServerError)
		info.Writer.Write([]byte("internal server error"))
	}()
	n := svr.router.Find(info.Request.Method, info.Request.URL.Path, info.UrlParams)
	if n == nil {
		_, err = svr.noRouteHandlers.Do(c, nil, info)
		return
	}
	info.PatternPath = n.fullPath
	evolog.GetLogContext(c).SetMethod(n.fullPath)
	_, err = n.handlers.Do(c, nil, info)
}

func (svr *Server) rebuild404Handlers() {
	if len(svr.userNoRouteHandlers) == 0 {
		svr.noRouteHandlers = append([]Handler{HandlerFunc(finalNoRouteHandler)}, svr.RouterBuilder.handlers...)
	} else {
		svr.noRouteHandlers = joinSlice(svr.RouterBuilder.handlers, svr.userNoRouteHandlers)
	}
}

type Handler = base.Interceptor[*RequestInfo]

type HandlerChain = base.Chain[*RequestInfo]

type HandlerFunc = base.InterceptorFunc[*RequestInfo]

func finalNoRouteHandler(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	resp, err = nexter.Next(c, req, info)
	if info.Writer.GetStatus() == 0 {
		info.Writer.WriteHeader(http.StatusNotFound)
	}
	return
}
