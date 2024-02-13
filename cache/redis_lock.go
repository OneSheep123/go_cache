package cache

import (
	"context"
	_ "embed"
	"errors"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"time"
)

var (
	ErrFailedToPreemptLock = errors.New("redis-lock: 抢锁失败")

	ErrLockNotHold = errors.New("redis-lock: 你没有持有锁")

	//go:embed lua/unlock.lua
	unLockLua string
)

type Client struct {
	client redis.Cmdable
}

func NewClient(client redis.Cmdable) *Client {
	return &Client{
		client: client,
	}
}

func (c *Client) TryLock(ctx context.Context, key string, expireTime time.Duration) (*Lock, error) {
	val := uuid.New().String()
	ok, err := c.client.SetNX(ctx, key, val, expireTime).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrFailedToPreemptLock
	}
	return &Lock{
		c:     c.client,
		key:   key,
		value: val,
	}, nil
}

type Lock struct {
	c     redis.Cmdable
	key   string
	value string
}

func (l *Lock) Unlock(ctx context.Context) error {
	// 使用lua脚本
	res, err := l.c.Eval(ctx, unLockLua, []string{l.key}, l.value).Int64()
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrLockNotHold
	}
	return nil
}
