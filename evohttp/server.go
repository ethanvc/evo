package evohttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/evolog"
	"golang.org/x/exp/slog"
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
	c = evolog.WithLogContext(c, req.Header.Get("x-trace-id"))
	n := svr.router.Find(req.Method, req.URL.Path, info.params)
	if n == nil {
		svr.serveHandlerNotFound(c, info)
		return
	}
	info.handlers = n.handlers
	info.PatternPath = n.fullPath
	evolog.GetLogContext(c).SetMethod(n.fullPath)
	handlerReq, err := svr.parserRequest(info)
	if err != nil {
		svr.writeResponse(info, 0, err, nil)
		return
	}
	handlerResp, err := info.Next(c, handlerReq)
	if _, ok := info.Handler().(*StdHandler); ok {
		slog.InfoContext(c, "REQ_LOG", slog.Any("req", handlerReq), slog.Any("resp", handlerResp), slog.Any("err", err))
	}
	svr.writeResponse(info, 0, err, handlerResp)
}

func (svr *Server) writeResponse(info *RequestInfo, code int, err error, data any) {
	if info.Writer.GetStatus() != 0 {
		return
	}
	var httpResp HttpResp
	s := base.StatusFromError(err)
	httpResp.Code = s.GetCode()
	httpResp.Msg = s.GetMsg()
	httpResp.Data = data
	info.Writer.WriteHeader(http.StatusOK)
	buf, _ := json.Marshal(&httpResp)
	info.Writer.Write(buf)
}

func (svr *Server) parserRequest(info *RequestInfo) (any, error) {
	if info.Request.Header.Get("content-type") != "application/json" {
		return nil, nil
	}
	h := info.Handler()
	if h == nil {
		return nil, nil
	}
	realH, _ := h.(*StdHandler)
	if realH == nil {
		return nil, nil
	}
	req := realH.NewReq()
	if req == nil {
		return nil, nil
	}
	buf, err := io.ReadAll(info.Request.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, req)
	if err != nil {
		return nil, err
	}
	return req, nil
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
