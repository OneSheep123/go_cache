package lock

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"time"
)

var (
	ErrFailedToPreemptLock = errors.New("redis-lock: 抢锁失败")

	ErrLockNotHold = errors.New("redis-lock: 你没有持有锁")

	//go:embed lua/unlock.lua
	unLockLua string

	//go:embed lua/lock.lua
	lockLua string

	//go:embed lua/refresh.lua
	refreshLua string
)

type Client struct {
	client redis.Cmdable
}

func NewClient(client redis.Cmdable) *Client {
	return &Client{
		client: client,
	}
}

// TryLock 尝试加锁
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
		key:        key,
		val:        val,
		client:     c.client,
		expiration: expiration,
		stopCh:     make(chan struct{}),
	}, nil
}

func (c *Client) Lock(ctx context.Context,
	key string,
	expiration time.Duration,
	timeout time.Duration,
	retry RetryStrategy,
) (*Lock, error) {
	var t *time.Timer
	val := uuid.NewString()

	for {
		timeoutCtx, cancelFunc := context.WithTimeout(ctx, timeout)
		result, err := c.client.Eval(timeoutCtx, lockLua, []string{key}, val, expiration).Result()
		cancelFunc()
		// 当前非超时的错误
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if result == "OK" {
			return &Lock{
				key:        key,
				val:        val,
				client:     c.client,
				expiration: expiration,
				stopCh:     make(chan struct{}),
			}, nil
		}
		duration, ok := retry.Next()
		if !ok {
			return nil, fmt.Errorf("redis-lock: 超出重试限制, %w", ErrFailedToPreemptLock)
		}
		if t == nil {
			t = time.NewTimer(duration)
		} else {
			t.Reset(duration)
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

type Lock struct {
	key        string
	val        string
	client     redis.Cmdable
	expiration time.Duration
	stopCh     chan struct{}
}

func (l *Lock) Unlock(ctx context.Context) error {
	res, err := l.client.Eval(ctx, unLockLua, []string{l.key}, l.val).Result()
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrLockNotHold
	}
	defer func() {
		select {
		case l.stopCh <- struct{}{}:
		default:
		}
	}()
	return nil
}

func (l *Lock) Refresh(ctx context.Context) error {
	result, err := l.client.Eval(ctx, refreshLua, []string{l.key}, l.val, l.expiration).Result()
	if err != nil {
		return err
	}
	if result != 1 {
		return ErrLockNotHold
	}
	return nil
}

func (l *Lock) AutoRefresh(interval time.Duration, timeout time.Duration) error {
	ticker := time.NewTicker(interval)
	timeoutCh := make(chan struct{}, 1)
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
