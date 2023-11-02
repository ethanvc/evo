package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
	"net/http"
)

type Server struct {
	*RouterBuilder
	noRouteChain     HandlerChain
	userNoRouteChain HandlerChain
}

func NewServer() *Server {
	svr := &Server{
		noRouteChain: []Handler{HandlerFunc(defaultNoRouteHandler)},
	}
	svr.RouterBuilder = NewRouterBuilder()
	svr.Use(NewLogHandler(), NewCodecHandler(), NewRecoverHandler())
	return svr
}

func (svr *Server) Use(handlers ...Handler) {
	svr.RouterBuilder.Use(handlers...)
	svr.rebuildNoRouteHandlers()
}

func (svr *Server) UseF(handlers ...HandlerFunc) {
	svr.Use(funcToHandlers(handlers)...)
}

func (svr *Server) NoRoute(handler ...Handler) {
	svr.userNoRouteChain = append(svr.userNoRouteChain, handler...)
	svr.rebuildNoRouteHandlers()
}

func (svr *Server) Run(addr string) {
	err := http.ListenAndServe(addr, http.HandlerFunc(svr.ServeHTTP))
	if err != nil {
		panic(err)
	}
}

func (svr *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	params := make(map[string]string)
	n := svr.FindRoute(req.Method, req.URL.Path, params)
	svr.ServeHTTPRouted(n, params, w, req)
}

func (svr *Server) FindRoute(method, urlPath string, params map[string]string) *RouteNode {
	return svr.router.Find(method, urlPath, params)
}

func (svr *Server) ServeHTTPRouted(n *RouteNode, params map[string]string, w http.ResponseWriter, req *http.Request) {
	info := NewRequestInfo()
	info.Request = req
	info.Writer.Reset(w)
	info.UrlParams = params
	c := context.WithValue(req.Context(), contextKeyRequestInfo{}, info)
	var h HandlerChain
	if n != nil {
		h = n.Handlers
		info.PatternPath = n.FullPath
	} else {
		h = svr.noRouteChain
	}
	h.Do(c, nil, info)
	if info.Writer.GetStatus() == 0 {
		info.Writer.WriteHeader(http.StatusInternalServerError)
		info.Writer.Write([]byte("internal server error"))
	}
}

func (svr *Server) rebuildNoRouteHandlers() {
	svr.noRouteChain = nil
	svr.noRouteChain = append(svr.noRouteChain, svr.RouterBuilder.handlers...)
	if len(svr.userNoRouteChain) == 0 {
		svr.noRouteChain = append(svr.noRouteChain, HandlerFunc(defaultNoRouteHandler))
	} else {
		svr.noRouteChain = append(svr.noRouteChain, svr.userNoRouteChain...)
	}
}

func FindGlobalHandler[T Handler](s *Server) T {
	for _, v := range s.RouterBuilder.handlers {
		vv, ok := v.(T)
		if ok {
			return vv
		}
	}
	var nilVal T
	return nilVal
}

type Handler = base.Interceptor[*RequestInfo]

type HandlerChain = base.Chain[*RequestInfo]

type HandlerFunc = base.InterceptorFunc[*RequestInfo]

func defaultNoRouteHandler(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	resp, err = nexter.Next(c, req, info)
	if info.Writer.GetStatus() == 0 {
		info.Writer.WriteHeader(http.StatusNotFound)
	}
	return
}
