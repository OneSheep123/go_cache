package lock

import "time"

type RetryStrategy interface {
	// 第一个返回值，重试的间隔
	// 第二个返回值，要不要继续重试
	Next() (time.Duration, bool)
}

type FixedIntervalRetryStrategy struct {
	Interval time.Duration
	MaxCnt   int8
	Cnt      int8
}

func (f *FixedIntervalRetryStrategy) Next() (time.Duration, bool) {
	f.Cnt++
	return f.Interval, f.Cnt > f.MaxCnt
}
