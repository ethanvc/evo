package redishelper

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
)

func Get[Resp any](c context.Context, cli redis.UniversalClient, key string) (*Resp, error) {
	cmd := cli.Get(c, key)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	var resp *Resp
	err := json.Unmarshal([]byte(cmd.String()), &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
