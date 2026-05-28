package queue

import (
	"sync"
	"testing"
)

// BenchmarkSwapQueue_Push-10    	 1792479	       632.7 ns/op
func BenchmarkSwapQueue_Push(b *testing.B) {
	q := NewSwapQueue[int](128, 1280)
	for i := 0; i < b.N; i++ {
		for cnt := 0; cnt < 10; cnt++ {
			_ = q.Push(i)
		}
	}
}

func TestSwapQueue_Basic(t *testing.T) {
	q := NewSwapQueue[int](10, 20)

	// 测试推入
	if err := q.Push(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := q.Push(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 测试消费
	var result []int
	q.Range(func(v int) {
		result = append(result, v)
	})

	if len(result) != 2 || result[0] != 1 || result[1] != 2 {
		t.Fatalf("expected [1, 2], got %v", result)
	}
}

func TestSwapQueue_Full(t *testing.T) {
	q := NewSwapQueue[int](2, 2)

	_ = q.Push(1)
	_ = q.Push(2)

	// 第三次推入应该报错
	err := q.Push(3)
	if err == nil {
		t.Fatal("expected error when queue is full, but got nil")
	}
}

func TestSwapQueue_Concurrency(t *testing.T) {
	q := NewSwapQueue[int](100, 1000)
	var wg sync.WaitGroup

	// 100 个生产者并发写入
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			_ = q.Push(val)
		}(i)
	}

	wg.Wait()

	// 消费者读取
	count := 0
	q.Range(func(v int) {
		count++
	})

	if count != 100 {
		t.Fatalf("expected 100 elements, got %d", count)
	}
}
