package evohttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/evolog"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc/codes"
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
	info.RequestTime = time.Now()
	info.Writer.Reset(w)
	c := context.WithValue(req.Context(), contextKeyRequestInfo{}, info)
	c = evolog.WithLogContext(c, req.Header.Get("x-trace-id"))
	svr.baseNext(c, info)
}

func (svr *Server) baseNext(c context.Context, info *RequestInfo) {
	resp, err := svr.panicNext(c, nil, info)
	info.FinishTime = time.Now()
	var logArgs []any
	if err != nil {
		logArgs = append(logArgs, slog.Any("err", err))
	}
	if info.ParsedRequest != nil {
		logArgs = append(logArgs, slog.Any("req", info.ParsedRequest))
	}
	if resp != nil {
		logArgs = append(logArgs, slog.Any("resp", resp))
	}
	logArgs = append(logArgs, slog.Duration("tc", info.FinishTime.Sub(info.RequestTime)))
	logArgs = append(logArgs, slog.String("path", info.PatternPath))
	slog.InfoContext(c, "REQ_END", logArgs...)
}

func (svr *Server) panicNext(c context.Context, req any, info *RequestInfo) (resp any, err error) {
	func() {
		panicked := true
		defer func() {
			if !panicked {
				return
			}
			r := recover()
			switch v := r.(type) {
			case *base.Status:
				err = v.Err()
			case base.StatusError:
				err = v
			case error:
				err = v
			default:
				err = base.New(codes.Internal, "UnknownErr").Err()
			}
		}()
		resp, err = svr.routeNext(c, req, info)
		panicked = false
	}()
	return
}

func (svr *Server) routeNext(c context.Context, req any, info *RequestInfo) (any, error) {
	n := svr.router.Find(info.Request.Method, info.Request.URL.Path, info.params)
	if n == nil {
		info.ResetHandlers(svr.noRouteHandlers)
		return info.Next(c, nil)
	}
	info.ResetHandlers(n.handlers)
	info.PatternPath = n.fullPath
	evolog.GetLogContext(c).SetMethod(n.fullPath)
	return svr.codecNext(c, req, info)
}

func (svr *Server) codecNext(c context.Context, req any, info *RequestInfo) (any, error) {
	stdH, ok := info.Handler().(*StdHandler)
	if !ok {
		return info.Next(c, req)
	}
	req = stdH.NewReq()
	info.ParsedRequest = req
	buf, err := io.ReadAll(info.Request.Body)
	if err != nil {
		return setStdResponse(info, err, nil)
	}
	if len(buf) > 0 {
		err = json.Unmarshal(buf, req)
		if err != nil {
			return setStdResponse(info, err, nil)
		}
	}
	resp, err := info.Next(c, req)
	return setStdResponse(info, err, resp)
}

func setStdResponse(info *RequestInfo, err error, data any) (any, error) {
	s := base.StatusFromError(err)
	info.Writer.Header().Set("content-type", "application/json")
	var httpResp HttpResp
	httpResp.Code = s.GetCode()
	httpResp.Msg = s.GetMsg()
	httpResp.Event = s.GetEvent()
	httpResp.Data = data
	buf, _ := json.Marshal(httpResp)
	info.Writer.Write(buf)
	return data, err
}

type Handler interface {
	HandleRequest(c context.Context, req any, info *RequestInfo) (any, error)
}

type HandlerChain []Handler

type HandlerFunc func(context.Context, any, *RequestInfo) (any, error)

func (h HandlerFunc) HandleRequest(c context.Context, req any, info *RequestInfo) (any, error) {
	return h(c, req, info)
}
