// create by chencanhua in 2023/9/10
package channel

import (
	"context"
)

type Task func()

// TaskPool 该任务 池允许开发者提交任务，并且设定最多多少个 goroutine 同时运行
type TaskPool struct {
	tasks chan Task
	close chan struct{}
}

func NewTaskPool(numG int, cap int) *TaskPool {
	t := &TaskPool{
		tasks: make(chan Task, cap),
		close: make(chan struct{}),
	}

	// todo: 对于慢任务和快任务的处理
	for i := 0; i < numG; i++ {
		go func() {
			for {
				select {
				case f := <-t.tasks:
					f()
				case <-t.close:
					return
				}
			}
		}()
	}

	return t
}

func (tp *TaskPool) Submit(ctx context.Context, t Task) error {
	select {
	case tp.tasks <- t:
	// 超时控制，防止任务队列满了情况
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// Close 要暴露出来
func (tp *TaskPool) Close() error {
	close(tp.close)
	return nil
}
