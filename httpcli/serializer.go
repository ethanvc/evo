package httpcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	err := json.Unmarshal(cliResp.Body, resp)
	if err != nil {
		return fmt.Errorf("unmarshal error: %s. body is %s", err.Error(), string(cliResp.Body))
	}
	return nil
}
