// create by chencanhua in 2023/9/15
package demo

import (
	"fmt"
	"golang.org/x/sync/singleflight"
	"testing"
)

// singleFlight 被包装的方法在对于同一个key，只会有一个协程执行，其他协程等待那个协程执行结束后，拿到同样的结果
// 可用于防止缓存击穿
func TestSingleFlight(t *testing.T) {
	g := new(singleflight.Group)

	block := make(chan struct{})
	res1c := g.DoChan("key", func() (interface{}, error) {
		<-block
		fmt.Println(12)
		return "func 1", nil
	})
	res2c := g.DoChan("key", func() (interface{}, error) {
		<-block
		fmt.Println(13)
		return "func 2", nil
	})
	close(block)

	res1 := <-res1c
	res2 := <-res2c

	// Results are shared by functions executed with duplicate keys.
	fmt.Println("Shared:", res2.Shared)
	// Only the first function is executed: it is registered and started with "key",
	// and doesn't complete before the second funtion is registered with a duplicate key.
	fmt.Println("Equal results:", res1.Val.(string) == res2.Val.(string))
	fmt.Println("Result:", res1.Val)

	// Output:
	// Shared: true
	// Equal results: true
	// Result: func 1
}
