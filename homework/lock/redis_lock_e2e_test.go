package lock

import (
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_e2e_TryLock(t *testing.T) {
	rdb := getRedisCmd()
	testCases := []struct {
		name string

		before func(t *testing.T)
		after  func(t *testing.T)

		key        string
		expiration time.Duration

		wantLock *Lock
		wantErr  error
	}{
		{
			name: "key exist",
			before: func(t *testing.T) {
				res, err := rdb.SetNX(context.Background(), "key1", 123, time.Minute).Result()
				require.NoError(t, err)
				assert.Equal(t, true, res)
			},
			after: func(t *testing.T) {
				res, err := rdb.Del(context.Background(), "key1").Result()
				require.NoError(t, err)
				assert.Equal(t, int64(1), res)
			},
			key:      "key1",
			wantLock: nil,
			wantErr:  ErrFailedToPreemptLock,
		},
		{
			name: "lock",
			after: func(t *testing.T) {
				res, err := rdb.Del(context.Background(), "key1").Result()
				require.NoError(t, err)
				assert.Equal(t, int64(1), res)
			},
			key: "key1",
			wantLock: &Lock{
				key: "key1",
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(rdb)
			tc.before(t)
			defer tc.after(t)
			lock, err := client.TryLock(context.Background(), tc.key, 10*time.Second)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantLock.key, lock.key)
			assert.NotEmpty(t, lock.val)
		})
	}
}

func getRedisCmd() redis.Cmdable {
	client := redis.NewClient(&redis.Options{
		Addr: "172.20.10.14:6379",
	})
	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	result, err := client.Ping(ctx).Result()
	cancelFunc()
	if err != nil {
		return nil
	}
	if result == "PONG" {
		return client
	}
	return nil
}
