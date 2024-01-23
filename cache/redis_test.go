// create by chencanhua in 2023/9/14
package cache

import (
	"context"
	"fmt"
	"geek_cache/cache/mocks"
	"geek_cache/internal/errs"
	"github.com/go-redis/redis/v9"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"testing"
	"time"
)

// 单元测试

func TestRedisCache_Set(t *testing.T) {
	// mockgen -destination=cache/mocks/mock_redis_cmdable.gen.go -package=mocks github.com/go-redis/redis/v9 Cmdable
	testCase := []struct {
		name       string
		key        string
		value      any
		expireTime time.Duration
		mock       func(ctrl *gomock.Controller) redis.Cmdable
		wantErr    error
	}{
		{
			name:       "set value",
			key:        "key1",
			value:      "value1",
			expireTime: time.Second,
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				// 刚刚生成的mocks包下的文件
				// 模拟redis操作
				cmd := mocks.NewMockCmdable(ctrl)
				statusCmd := redis.NewStatusCmd(context.Background())
				statusCmd.SetVal("OK")
				statusCmd.SetErr(nil)
				cmd.EXPECT().
					Set(context.Background(), "key1", "value1", time.Second).
					Return(statusCmd)
				return cmd
			},
		},
		{
			name:       "timeout",
			key:        "key1",
			value:      "value1",
			expireTime: time.Second,
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				// 刚刚生成的mocks包下的文件
				// 模拟redis操作
				cmd := mocks.NewMockCmdable(ctrl)
				statusCmd := redis.NewStatusCmd(context.Background())
				statusCmd.SetErr(context.DeadlineExceeded)
				cmd.EXPECT().
					Set(context.Background(), "key1", "value1", time.Second).
					Return(statusCmd)
				return cmd
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name:       "NO OK",
			key:        "key1",
			value:      "value1",
			expireTime: time.Second,
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				// 刚刚生成的mocks包下的文件
				// 模拟redis操作
				cmd := mocks.NewMockCmdable(ctrl)
				statusCmd := redis.NewStatusCmd(context.Background())
				statusCmd.SetVal("NOT OK")
				cmd.EXPECT().
					Set(context.Background(), "key1", "value1", time.Second).
					Return(statusCmd)
				return cmd
			},
			wantErr: fmt.Errorf("%w, 返回信息 %s", errs.ErrFailedToSetCache, "NOT OK"),
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			c := NewRedisCache(tc.mock(ctrl))
			err := c.Set(context.Background(), tc.key, tc.value, tc.expireTime)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestRedisCache_Get(t *testing.T) {
	testCases := []struct {
		name    string
		key     string
		wantVal any
		wantErr error
		mock    func(ctrl *gomock.Controller) redis.Cmdable
	}{
		{
			name:    "val1",
			key:     "key1",
			wantVal: "val1",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				result := redis.NewStringResult("val1", nil)
				cmd.EXPECT().
					Get(context.Background(), "key1").
					Return(result)
				return cmd
			},
			wantErr: nil,
		},
		{
			name: "timeout",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				str := redis.NewStringCmd(context.Background())
				str.SetErr(context.DeadlineExceeded)
				cmd.EXPECT().
					Get(context.Background(), "key1").
					Return(str)
				return cmd
			},
			key:     "key1",
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			redisCache := NewRedisCache(tc.mock(ctrl))
			resVal, err := redisCache.Get(context.Background(), tc.key)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantVal, resVal)
		})
	}
}

func TestReadThrough_Get(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		wantValue   any
		readThrough func(ca Cache) *ReadThrough
		mock        func(ctrl *gomock.Controller, key string) redis.Cmdable
		wantErr     error
	}{
		{
			name:      "key1",
			key:       "key1",
			wantValue: 12,
			readThrough: func(ca Cache) *ReadThrough {
				return &ReadThrough{
					Cache: ca,
					// 从DB拿数据
					LoadFunc: func(ctx context.Context, key string) (any, error) {
						log.Printf("从数据库拿了:%s 的数据\n", key)
						return 12, nil
					},
					ExpireTime: 12 * time.Second,
				}
			},
			mock: func(ctrl *gomock.Controller, key string) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				stringCmd := redis.NewStringCmd(context.Background())
				stringCmd.SetErr(errs.ErrKeyNotFound)
				// 模拟Redis的Get操作
				cmd.EXPECT().Get(context.Background(), key).Return(stringCmd)

				statusCmd := redis.NewStatusCmd(context.Background())
				statusCmd.SetVal("OK")
				// 模拟Redis的Set操作
				cmd.EXPECT().Set(context.Background(), key, 12, 12*time.Second).Return(statusCmd)
				return cmd
			},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			redisCache := NewRedisCache(tc.mock(ctrl, tc.key))
			// 封装缓存的使用，使用读穿透模式
			readThrough := tc.readThrough(redisCache)
			value, err := readThrough.Get(context.Background(), tc.key)
			require.NoError(t, err)
			assert.Equal(t, tc.wantValue, value)
		})
	}
}
