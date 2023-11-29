// create by chencanhua in 2023/9/12
package cache

import (
	"context"
	"geek_cache/internal/errs"
	"sync/atomic"
	"time"
)

type MaxCntCache struct {
	*BuildInMapCache
	cnt    int32
	maxCnt int32
}

func BuildMaxCntCache(b *BuildInMapCache, maxCnt int32) *MaxCntCache {
	res := &MaxCntCache{
		BuildInMapCache: b,
		maxCnt:          maxCnt,
	}

	origin := b.onEvicted

	// 在原有的onEvicted上，再次进行封装onEvicted，用于cnt--
	res.onEvicted = func(key string, value any) {
		atomic.AddInt32(&res.cnt, -1)
		origin(key, value)
	}

	return res
}

// Set 重写localCache中的set方法，用于cnt计数++
func (m *MaxCntCache) Set(ctx context.Context, key string, value any, expireTime time.Duration) error {

	// 1. 这种写法，如果 key 已经存在，你这计数就不准了
	//cnt := atomic.AddInt32(&c.cnt, 1)
	//if cnt > c.maxCnt {
	//	atomic.AddInt32(&c.cnt, -1)
	//	return errOverCapacity
	//}
	//return c.BuildInMapCache.Set(ctx, key, val, expiration)

	// 2. 这种写法，当mutex被解锁时候，锁被别人抢到且还未执行set时，cnt会被出现再加情况
	//c.mutex.Lock()
	//_, ok := c.data[key]
	//if !ok {
	//	c.cnt ++
	//}
	//if c.cnt > c.maxCnt {
	//	c.mutex.Unlock()
	//	return errOverCapacity
	//}
	//c.mutex.Unlock()
	//return c.BuildInMapCache.Set(ctx, key, val, expiration)

	m.mutex.Lock()
	// 这里锁住下面的set了
	defer m.mutex.Unlock()
	_, ok := m.m[key]
	if !ok {
		if m.cnt+1 > m.maxCnt {
			return errs.ErrOverCapacity
		}
		m.cnt++
	}
	return m.set(ctx, key, value, expireTime)
}
