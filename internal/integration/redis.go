package integration

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedis(addr string) *RedisClient {
	return &RedisClient{
		Client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *RedisClient) SetPX(ctx context.Context, key string, value string, px time.Duration) error {
	return r.Client.Set(ctx, key, value, px).Err()
}
