package rediscli

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisClient struct {
	cli  redis.UniversalClient
	conf *RedisClientConfig
}

func NewRedisClient(cli redis.UniversalClient, conf *RedisClientConfig) *RedisClient {
	if conf == nil {
		conf = &RedisClientConfig{}
	}
	return &RedisClient{
		cli:  cli,
		conf: conf,
	}
}

func (cli *RedisClient) Get(c context.Context, key string, resp any) error {
	cmd := cli.cli.Get(c, key)
	if cmd.Err() == nil {
		return cli.decode(c, key, cmd, resp)
	} else {
		return cmd.Err()
	}
}

func (cli *RedisClient) Set(c context.Context, key string, value any, expire time.Duration) error {
	realVal, err := cli.encode(c, key, value)
	if err != nil {
		return err
	}
	cmd := cli.cli.Set(c, key, realVal, expire)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func (cli *RedisClient) decode(c context.Context, key string, cmd *redis.StringCmd, resp any) error {
	if cli.conf.Decoder != nil {
		return cli.conf.Decoder(c, key, cmd, resp)
	}
	buf, err := cmd.Bytes()
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, resp)
	if err != nil {
		return err
	}
	return nil
}

func (cli *RedisClient) encode(c context.Context, key string, value any) (any, error) {
	if cli.conf.Encoder != nil {
		return cli.conf.Encoder(c, key, value)
	}
	buf, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func IsBasicType(v any) bool {
	if v == nil {
		return true
	}
	return false
}

type RedisClientConfig struct {
	Encoder func(c context.Context, key string, value any) (any, error)
	Decoder func(c context.Context, key string, cmd *redis.StringCmd, resp any) error
}
