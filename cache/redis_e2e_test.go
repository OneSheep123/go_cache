//go:build e2e

// create by chencanhua in 2023/9/15
package cache

import (
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRedisCache_e2e_Set(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	redisClient := NewRedisCache(client)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()
	key := "test-key"
	value := 12
	err := redisClient.Set(ctx, key, value, 10*time.Second)
	require.NoError(t, err)
	res, err := client.Get(ctx, key).Int()
	require.NoError(t, err)
	assert.Equal(t, value, res)
}

func TestRedisCache_e2e_SetV1(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	testCase := []struct {
		name  string
		after func(t *testing.T)

		key        string
		value      string
		expiration time.Duration

		wantErr error
	}{
		{
			name: "set value",
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				res, err := rdb.Get(ctx, "key1").Result()
				require.NoError(t, err)
				assert.Equal(t, "value1", res)
				_, err = rdb.Del(ctx, "key1").Result()
				require.NoError(t, err)
			},
			key:        "key1",
			value:      "value1",
			expiration: time.Minute,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			c := NewRedisCache(rdb)
			//tc.before()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err := c.Set(ctx, tc.key, tc.value, tc.expiration)
			require.NoError(t, err)
			tc.after(t)
		})
	}
}
