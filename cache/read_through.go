// create by chencanhua in 2023/9/15
package cache

import (
	"context"
	"errors"
	"fmt"
	"geek_cache/internal/errs"
	"golang.org/x/sync/singleflight"
	"log"
	"time"
)

var (
	ErrFailedToRefreshCache = errors.New("刷新缓存失败")
)

type ReadThrough struct {
	Cache
	LoadFunc   func(ctx context.Context, key string) (any, error)
	ExpireTime time.Duration
}

// Get 读穿透
func (r *ReadThrough) Get(ctx context.Context, key string) (any, error) {
	val, err := r.Cache.Get(ctx, key)
	if err == errs.ErrKeyNotFound {
		val, err = r.LoadFunc(ctx, key)
		if err == nil {
			er := r.Cache.Set(ctx, key, val, r.ExpireTime)
			if er != nil {
				return val, fmt.Errorf("%w, 原因：%s", ErrFailedToRefreshCache, er.Error())
			}
		}
	}
	return val, err
}

// GetAsync 读穿透(异步)
func (r *ReadThrough) GetAsync(ctx context.Context, key string) (any, error) {
	val, err := r.Cache.Get(ctx, key)
	if err == errs.ErrKeyNotFound {
		go func() {
			val, err = r.LoadFunc(ctx, key)
			if err == nil {
				er := r.Cache.Set(ctx, key, val, r.ExpireTime)
				if er != nil {
					log.Fatal(er)
				}
			}
		}()
	}
	return val, err
}

// GetSemiAsync 读穿透(半异步)
func (r *ReadThrough) GetSemiAsync(ctx context.Context, key string) (any, error) {
	val, err := r.Cache.Get(ctx, key)
	if err == errs.ErrKeyNotFound {
		val, err = r.LoadFunc(ctx, key)
		go func() {
			if err == nil {
				er := r.Cache.Set(ctx, key, val, r.ExpireTime)
				if er != nil {
					log.Fatal(er)
				}
			}
		}()
	}
	return val, err
}

type ReadThroughV1[T any] struct {
	Cache
	LoadFunc   func(ctx context.Context, key string) (any, error)
	ExpireTime time.Duration
	g          singleflight.Group
}

func (r *ReadThroughV1[T]) Get(ctx context.Context, key string) (T, error) {
	val, err := r.Cache.Get(ctx, key)
	if err == errs.ErrKeyNotFound {
		val, err, _ = r.g.Do(key, func() (interface{}, error) {
			tempVal, err := r.LoadFunc(ctx, key)
			if err == nil {
				er := r.Cache.Set(ctx, key, val, r.ExpireTime)
				if er != nil {
					return val, fmt.Errorf("%w, 原因：%s", ErrFailedToRefreshCache, er.Error())
				}
			}
			return tempVal, err
		})
	}
	return val.(T), err
}
