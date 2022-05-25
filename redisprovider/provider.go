package redisprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

const redisNil = "redis: nil"

type RedisProvider struct {
	client *redis.Client
}

func New(options *redis.Options) (*RedisProvider, error) {
	client := redis.NewClient(options)

	if _, err := client.Ping().Result(); err != nil {
		return nil, fmt.Errorf("redis.Ping(): %w", err)
	}

	return &RedisProvider{client: client}, nil
}

func (p *RedisProvider) Get(_ context.Context, key string) ([]byte, error) {
	value, err := p.client.Get(key).Result()
	if err != nil {
		if err.Error() == redisNil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis.Get(): %w", err)
	}

	if value == "" || value == redisNil {
		return nil, nil
	}

	return []byte(value), nil
}

func (p *RedisProvider) Set(_ context.Context, key string, value []byte, expiry time.Duration) error {
	cmd := p.client.Set(key, value, expiry)
	if err := cmd.Err(); err != nil {
		return fmt.Errorf("redis.Set(): %w", err)
	}
	return nil
}
