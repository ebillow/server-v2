package queue

import "testing"

var (
	q SwapQueue[int]
	c = make(chan int, 100)
)

// BenchmarkSwapQueue_Push-10    	 1792479	       632.7 ns/op
func BenchmarkSwapQueue_Push(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for cnt := 0; cnt < 10; cnt++ {
			_ = q.PushAndWake(i)
		}
	}
}
