package httpproto

import (
	"context"
	"encoding/json"

	"github.com/ethanvc/evo/httpcli"
	"github.com/ethanvc/evo/obs"
)

type Serializer struct {
	httpcli.JsonSerializer
}

func (s *Serializer) Unmarshal(ctx context.Context, cliResp *httpcli.CliResp, resp any, opts *httpcli.Options) error {
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
