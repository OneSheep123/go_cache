package channel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBroker(t *testing.T) {
	b := &Broker{}

	go func() {
		err := b.Send(Msg{Content: time.Now().String()})
		if err != nil {
			t.Log(err)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}()

	wg := sync.WaitGroup{}
	wg.Add(3)
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("消费者 %d", i)
		go func() {
			defer wg.Done()
			msgs, err := b.Subscribe(10)
			if err != nil {
				t.Log(err)
				return
			}

			for {
				select {
				case v := <-msgs:
					fmt.Println(name, v.Content)
				default:
					fmt.Println(name, "当前无值")
					time.Sleep(1 * time.Second)
				}
			}
		}()
	}

	wg.Wait()
}
