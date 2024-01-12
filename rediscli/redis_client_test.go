package rediscli

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"testing"
)

func newTestClient() *RedisClient {
	opt := &redis.UniversalOptions{}
	cli := redis.NewUniversalClient(opt)
	return NewRedisClient(cli, nil)
}

func TestRedisClient_Set(t *testing.T) {
	cli := newTestClient()
	err := cli.Set(context.Background(), "a", 3, 0)
	require.NoError(t, err)
	var n int
	err = cli.Get(context.Background(), "a", &n)
	require.NoError(t, err)
	require.Equal(t, 3, n)
}
