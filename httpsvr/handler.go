package httpsvr

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"time"
)

type Handler struct {
	Serializer   Serializer
	Timeout      time.Duration
	Interceptors []Interceptor
	reqType      reflect.Type
	f            reflect.Value
}

func NewHandler(f any) *Handler {
	hh := &Handler{}
	hh.init(f)
	return hh
}

func (h *Handler) init(f any) {
	if f == nil {
		panic("must not zero for handler func")
	}
	reqType, err := validateAndParseFunc(f)
	if err != nil {
		panic(err)
	}
	h.reqType = reqType
	h.f = reflect.ValueOf(f)
}

func (h *Handler) Handle(ctx context.Context, info *CallInfo) (err error) {
	var req, resp any
	ctx = info.Server.getLogger().Start(ctx, info)
	defer func() {
		info.Server.getLogger().End(ctx, err, req, resp, info)
	}()
	req, err = h.unmarshal(ctx, info)
	if err != nil {
		err = h.marshalAndWrite(ctx, err, nil, info)
		return err
	}
	resp, err = h.handleRest(ctx, req, info)
	err = h.marshalAndWrite(ctx, err, resp, info)
	return err
}

func (h *Handler) handleRest(ctx context.Context, req any, info *CallInfo) (any, error) {
	next := &Next{
		i:            0,
		handler:      h,
		interceptors: h.getInterceptors(info.Server),
	}
	return next.Next(ctx, req, info)
}

func (h *Handler) getInterceptors(s *Server) []Interceptor {
	if h.Interceptors != nil {
		return h.Interceptors
	}
	return s.Interceptors
}

func (h *Handler) unmarshal(ctx context.Context, info *CallInfo) (any, error) {
	v := reflect.New(h.reqType.Elem()).Interface()
	switch realV := v.(type) {
	case *io.ReadCloser:
		*realV = info.Request.Body
		return v, nil
	}
	buf, err := io.ReadAll(info.Request.Body)
	if err != nil {
		return nil, err
	}
	info.RequestBody = buf
	if len(buf) == 0 {
		return v, nil
	}
	switch realV := v.(type) {
	case *string:
		*realV = string(buf)
		return v, nil
	case *[]byte:
		*realV = buf
		return v, nil
	}
	serializer := h.getSerializer(info.Server)
	err = serializer.Unmarshal(ctx, v, info)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (h *Handler) marshalAndWrite(ctx context.Context, err error, resp any, info *CallInfo) error {
	responseBody, newErr := h.marshal(ctx, err, resp, info)
	if newErr != nil {
		info.StatusCode = http.StatusInternalServerError
		return newErr
	}
	info.Writer.WriteHeader(info.StatusCode)
	if responseBody != nil {
		_, newErr = io.Copy(info.Writer, responseBody)
		newErr2 := responseBody.Close()
		if newErr2 != nil {
			h.getLogger(info.Server).Log(ctx, slog.LevelError, "CloseResponseBodyErr", slog.Any("err", newErr2))
		}
		if newErr != nil {
			h.logWriteErr(ctx, info, newErr)
		}
	}
	return err
}

func (h *Handler) getLogger(s *Server) Logger {
	return s.getLogger()
}

func (h *Handler) logWriteErr(ctx context.Context, info *CallInfo, err error) {}

func (h *Handler) marshal(ctx context.Context, respErr error, resp any, info *CallInfo) (responseBody io.ReadCloser, err error) {
	s := h.getSerializer(info.Server)
	switch realV := resp.(type) {
	case *Empty:
		// let responseBody nil
	case *string:
		info.ResponseBody = []byte(*realV)
		responseBody = io.NopCloser(bytes.NewReader(info.ResponseBody))
	case *[]byte:
		info.ResponseBody = *realV
		responseBody = io.NopCloser(bytes.NewReader(info.ResponseBody))
	case *io.ReadCloser:
		responseBody = *realV
	default:
		var marshalErr error
		responseBody, marshalErr = s.Marshal(ctx, respErr, resp, info)
		if marshalErr != nil {
			info.StatusCode = http.StatusInternalServerError
			return nil, marshalErr
		}
	}

	if info.StatusCode == 0 {
		info.StatusCode = s.GetStatusCode(ctx, respErr)
	}
	if info.StatusCode == 0 {
		if respErr != nil {
			info.StatusCode = http.StatusBadRequest
		} else {
			info.StatusCode = http.StatusOK
		}
	}
	return responseBody, nil
}

func (h *Handler) getSerializer(s *Server) Serializer {
	if h.Serializer != nil {
		return h.Serializer
	}
	return s.getSerializer()
}

func (h *Handler) NameOfFunc() string {
	return nameOfFunction(h.f.Interface())
}

func (h *Handler) call(ctx context.Context, req any) (any, error) {
	if req == nil {
		return nil, fmt.Errorf("req must not nil when call handler")
	}
	if reflect.TypeOf(req) != h.reqType {
		return nil, fmt.Errorf("invalid req type, expect %v, got %v", h.reqType, reflect.TypeOf(req))
	}
	result := h.f.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req)})
	resp := result[0].Interface()

	var err error
	reflectErr := result[1].Interface()
	if reflectErr != nil {
		err = reflectErr.(error)
	}
	return resp, err
}

type Empty struct{}
