// create by chencanhua in 2023/9/14
package cache

import (
	"context"
	"fmt"
	"geek_cache/internal/errs"
	"github.com/go-redis/redis/v9"
	"time"
)

type RedisCache struct {
	client redis.Cmdable
}

func NewRedisCache(client redis.Cmdable) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

func (r *RedisCache) Get(ctx context.Context, key string) (any, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, expireTime time.Duration) error {
	result, err := r.client.Set(ctx, key, value, expireTime).Result()
	if err != nil {
		return err
	}
	if result != "OK" {
		return fmt.Errorf("%w, 返回信息 %s", errs.ErrFailedToSetCache, result)
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	_, err := r.client.Del(ctx, key).Result()
	return err
}

func (r *RedisCache) LoadAndDelete(ctx context.Context, key string) (any, error) {
	return r.client.GetDel(ctx, key).Result()
}
