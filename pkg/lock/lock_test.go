package lock

import (
	"sync"
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		mockLock(t)
		wg.Done()
	}()
	mockLock(t)
	wg.Wait()
}

func mockLock(t *testing.T) {
	mtx := NewLock("lock1")
	t.Log("start lock")
	err := mtx.Lock()
	t.Log("lock")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	mtx.Unlock()
	t.Log("unlock")
}

// BenchmarkLock-10    	    2580	    423525 ns/op
func BenchmarkLock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mtx := NewLock("lock1")
		err := mtx.Lock()
		if err != nil {
			b.Fatal(err)
		}
		mtx.Unlock()
	}
}
