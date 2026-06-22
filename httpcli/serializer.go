package httpcli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/ethanvc/evo/obs"
)

type Serializer interface {
	Marshal(ctx context.Context, v any, opts *Options) (string, io.Reader, error)
	Unmarshal(ctx context.Context, cliResp *CliResp, resp any, opts *Options) error
}

type JsonSerializer struct {
}

func (s *JsonSerializer) Marshal(ctx context.Context, req any, opts *Options) (string, io.Reader, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return "", nil, err
	}
	return "application/json; charset=utf-8", bytes.NewReader(buf), nil
}

func (s *JsonSerializer) Unmarshal(ctx context.Context, cliResp *CliResp, resp any, opts *Options) error {
	var dto ResponseDto
	dto.SetData(resp)
	err := json.Unmarshal(cliResp.Body, &dto)
	if err != nil {
		return err
	}
	if dto.GetCode() != 0 {
		return obs.New(obs.Code(dto.GetCode()), dto.GetMsg())
	}
	return nil
}
