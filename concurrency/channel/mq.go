package channel

import (
	"errors"
	"sync"
)

// Broker 基于内存的消息队列
type Broker struct {
	lock  sync.RWMutex
	chans []chan Msg
}

type Msg struct {
	Content string
}

func (b *Broker) Send(m Msg) error {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for _, c := range b.chans {
		select {
		case c <- m:
		default:
			return errors.New("消息队列已满")
		}
	}
	return nil
}

func (b *Broker) Close() {
	b.lock.Lock()
	chans := b.chans
	b.chans = nil
	for _, c := range chans {
		close(c)
	}
	b.lock.Unlock()
}

func (b *Broker) Subscribe(cap int) (<-chan Msg, error) {
	res := make(chan Msg, cap)
	b.lock.Lock()
	defer b.lock.Unlock()
	b.chans = append(b.chans, res)
	return res, nil
}
