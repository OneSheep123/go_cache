// create by chencanhua in 2024/1/24
package cache

import (
	"context"
	"fmt"
	"geek_cache/internal/errs"
)

// BloomFilterCache 支持注入布隆过滤器
// 非侵入式
type BloomFilterCache struct {
	ReadThrough
}

func NewBloomFilterCache(cache Cache, bf BloomFilter, LoadFunc func(ctx context.Context, key string) (any, error)) *BloomFilterCache {
	return &BloomFilterCache{
		ReadThrough: ReadThrough{
			Cache: cache,
			LoadFunc: func(ctx context.Context, key string) (any, error) {
				ok := bf.HasKey(ctx, key)
				if ok {
					return LoadFunc(ctx, key)
				}
				return nil, errs.ErrKeyNotFound
			},
		},
	}
}

type BloomFilter struct {
	HasKey func(ctx context.Context, key string) bool
}

// BloomFilterCacheV1 注入布隆过滤器, 侵入式写法
type BloomFilterCacheV1 struct {
	ReadThrough
	bf BloomFilter
}

func (b *BloomFilterCacheV1) Get(ctx context.Context, key string) (any, error) {
	data, err := b.Cache.Get(ctx, key)
	if err == errs.ErrKeyNotFound && b.bf.HasKey(ctx, key) {
		data, err = b.LoadFunc(ctx, key)
		if err == nil {
			if e := b.Cache.Set(ctx, key, data, b.ExpireTime); e != nil {
				return data, fmt.Errorf("%w, 原因：%s", ErrFailedToRefreshCache, e.Error())
			}
		}
	}
	return data, err
}
