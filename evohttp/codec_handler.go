package evohttp

import (
	"context"
	"encoding/json"
	"github.com/ethanvc/evo/base"
	"github.com/mitchellh/mapstructure"
	"io"
)

type codecHandler struct {
}

func NewCodecHandler() Handler {
	return &codecHandler{}
}

func (h *codecHandler) Handle(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (resp any, err error) {
	stdH, ok := nexter.LastHandler().(*StdHandler)
	if !ok {
		return nexter.Next(c, req, info)
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
	err = h.fillUrlParam(req, info.UrlParams)
	if err != nil {
		return setStdResponse(info, err, nil)
	}
	resp, err = nexter.Next(c, req, info)
	return setStdResponse(info, err, resp)
}

func (h *codecHandler) fillUrlParam(v any, params map[string]string) (err error) {
	if len(params) == 0 {
		return
	}
	fillConfig := &mapstructure.DecoderConfig{
		Squash:           true,
		TagName:          "json",
		Result:           v,
		WeaklyTypedInput: true,
	}
	coder, err := mapstructure.NewDecoder(fillConfig)
	if err != nil {
		return
	}
	err = coder.Decode(params)
	if err != nil {
		return
	}
	return
}

func setStdResponse(info *RequestInfo, err error, data any) (any, error) {
	if info.Writer.GetStatus() != 0 {
		return data, err
	}
	s := base.StatusFromError(err)
	info.Writer.Header().Set("content-type", "application/json")
	var httpResp HttpResp[any]
	httpResp.Code = s.GetCode()
	httpResp.Msg = s.GetMsg()
	httpResp.Event = s.GetEvent()
	httpResp.Data = data
	buf, _ := json.Marshal(httpResp)
	info.Writer.Write(buf)
	return data, err
}
