// create by chencanhua in 2023/11/28
package errs

import "errors"

var (
	ErrKeyNotFound      = errors.New("cache：键不存在")
	ErrOverCapacity     = errors.New("cache：超过容量限制")
	ErrFailedToSetCache = errors.New("cache: 写入 redis 失败")
)
