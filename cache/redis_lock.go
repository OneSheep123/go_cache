package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
	"time"
)

var (
	ErrFailedToPreemptLock = errors.New("redis-lock: 抢锁失败")

	ErrLockNotHold = errors.New("redis-lock: 你没有持有锁")

	//go:embed lua/unlock.lua
	unLockLua string
	//go:embed lua/refresh.lua
	refreshLua string
	//go:embed lua/lock.lua
	lockLua string
)

type Client struct {
	client redis.Cmdable
	g      singleflight.Group
}

func NewClient(client redis.Cmdable) *Client {
	return &Client{
		client: client,
		g:      singleflight.Group{},
	}
}

func (c *Client) SinglefightLock(ctx context.Context,
	key string,
	expiration time.Duration,
	timeout time.Duration,
	retry RetryStrategy) (*Lock, error) {
	for {
		flag := false
		resChan := c.g.DoChan(key, func() (interface{}, error) {
			// 只有一个goroutine会执行到这里
			flag = true
			return c.Lock(ctx, key, expiration, timeout, retry), nil
		})
		select {
		case res := <-resChan:
			if flag {
				c.g.Forget(key)
				if res.Err != nil {
					return nil, res.Err
				}
				return res.Val.(*Lock), nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Lock 支持重试上锁
// ctx 可以控制总共时长 key 上锁的key expiration 锁时长 timeout 重试合计超时时间 retry 重试迭代器
func (c *Client) Lock(ctx context.Context,
	key string,
	expiration time.Duration,
	timeout time.Duration,
	retry RetryStrategy) (*Lock, error) {
	var timer *time.Timer
	val := uuid.New().String()
	for {
		lctx, cancelFunc := context.WithTimeout(ctx, timeout)
		res, err := c.client.Eval(lctx, lockLua, []string{key}, val, expiration.Seconds()).Result()
		cancelFunc()
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}

		if res == "OK" {
			return &Lock{
				key:        key,
				value:      val,
				c:          c.client,
				expiration: expiration,
				stopCh:     make(chan struct{}, 1),
			}, nil
		}
		interval, ok := retry.Next()
		if !ok {
			return nil, fmt.Errorf("redis-lock: 超出重试限制, %w", ErrFailedToPreemptLock)
		}
		if timer == nil {
			timer = time.NewTimer(interval)
		} else {
			timer.Reset(interval)
		}
		select {
		case <-timer.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (c *Client) TryLock(ctx context.Context, key string, expiration time.Duration) (*Lock, error) {
	val := uuid.New().String()
	ok, err := c.client.SetNX(ctx, key, val, expiration).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrFailedToPreemptLock
	}
	return &Lock{
		c:          c.client,
		key:        key,
		value:      val,
		expiration: expiration,
		stopCh:     make(chan struct{}, 1),
	}, nil
}

type Lock struct {
	c          redis.Cmdable
	key        string
	value      string
	expiration time.Duration
	stopCh     chan struct{}
}

func (l *Lock) Unlock(ctx context.Context) error {
	// 使用lua脚本
	res, err := l.c.Eval(ctx, unLockLua, []string{l.key}, l.value).Int64()
	defer func() {
		select {
		case l.stopCh <- struct{}{}:
		default:
			// 说明没有人调用 AutoRefresh
		}
	}()
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrLockNotHold
	}
	return nil
}

func (l *Lock) Refresh(ctx context.Context) error {
	// 使用lua脚本
	res, err := l.c.Eval(ctx, refreshLua, []string{l.key}, l.value, l.expiration.Seconds()).Int64()
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrLockNotHold
	}
	return nil
}

// AutoRefresh 自动续期
func (l *Lock) AutoRefresh(interval time.Duration, timeout time.Duration) error {
	timeoutCh := make(chan struct{}, 1)
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
			err := l.Refresh(ctx)
			cancelFunc()
			if errors.Is(err, context.DeadlineExceeded) {
				timeoutCh <- struct{}{}
				continue
			}
			if err != nil {
				return err
			}
		case <-timeoutCh:
			ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
			err := l.Refresh(ctx)
			cancelFunc()
			if errors.Is(err, context.DeadlineExceeded) {
				timeoutCh <- struct{}{}
				continue
			}
			if err != nil {
				return err
			}
		case <-l.stopCh:
			return nil
		}
	}
}

// 使用方法
// go l.AutoRefresh(1*time.Second, 10*time.Second)
