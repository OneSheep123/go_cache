// create by chencanhua in 2023/9/15
package cache

import (
	"golang.org/x/net/context"
	"log"
	"time"
)

type WriteThrough struct {
	Cache
	StoreFunc func(ctx context.Context, key string, value any, expireTime time.Duration) error
}

func (w *WriteThrough) Set(ctx context.Context, key string, value any, expireTime time.Duration) error {
	err := w.Cache.Set(ctx, key, value, expireTime)
	if err != nil {
		return err
	}
	return w.StoreFunc(ctx, key, value, expireTime)
}

func (w *WriteThrough) SetSemiAsync(ctx context.Context, key string, value any, expireTime time.Duration) error {
	err := w.Cache.Set(ctx, key, value, expireTime)
	go func() {
		er := w.StoreFunc(ctx, key, value, expireTime)
		if er != nil {
			log.Fatalln(er)
		}
	}()
	return err
}

func (w *WriteThrough) SetAsync(ctx context.Context, key string, value any, expireTime time.Duration) error {
	go func() {
		err := w.StoreFunc(ctx, key, value, expireTime)
		if err != nil {
			log.Fatal(err)
		}
		err = w.Cache.Set(ctx, key, value, expireTime)
		if err != nil {
			log.Fatalln(err)
		}
	}()
	return nil
}
