// create by chencanhua in 2024/1/24
package cache

import (
	"context"
	"fmt"
	"geek_cache/internal/errs"
	"golang.org/x/sync/singleflight"
)

// SingleflightCacheV1 装饰器模式
// 进一步封装 ReadThrough
// 非侵入式
type SingleflightCacheV1 struct {
	ReadThrough
}

func NewSinglflightCache(cache Cache, loadFunc func(ctx context.Context, key string) (any, error)) *SingleflightCacheV1 {
	g := &singleflight.Group{}
	return &SingleflightCacheV1{
		ReadThrough: ReadThrough{
			Cache: cache,
			LoadFunc: func(ctx context.Context, key string) (any, error) {
				v, err, _ := g.Do(key, func() (interface{}, error) {
					return loadFunc(ctx, key)
				})
				return v, err
			},
		},
	}
}

// SingleflightCacheV2 侵入式的方法
type SingleflightCacheV2 struct {
	ReadThrough
	g singleflight.Group
}

func (r *SingleflightCacheV2) Get(ctx context.Context, key string) (any, error) {
	data, err := r.Cache.Get(ctx, key)
	if err == errs.ErrKeyNotFound {
		data, err, _ = r.g.Do(key, func() (interface{}, error) {
			v, er := r.LoadFunc(ctx, key)
			if er == nil {
				er = r.Cache.Set(ctx, key, data, r.ExpireTime)
				if er != nil {
					return v, fmt.Errorf("%w, 原因：%s", ErrFailedToRefreshCache, er.Error())
				}
			}
			return v, er
		})
	}
	return data, err
}
