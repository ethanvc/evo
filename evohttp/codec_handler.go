package evohttp

import (
	"context"
	"encoding/json"
	"github.com/ethanvc/evo/base"
	"io"
)

type codecHandler struct {
}

func NewCodecHandler() Handler {
	return &codecHandler{}
}

func (h *codecHandler) HandleRequest(c context.Context, req any, info *RequestInfo) (resp any, err error) {
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
	resp, err = info.Next(c, req)
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
