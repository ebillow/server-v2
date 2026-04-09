package lock

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	err := mtx.Lock(context.Background())
	t.Log("lock")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	mtx.Unlock(context.Background())
	t.Log("unlock")
}

// BenchmarkLock-10    	    2580	    423525 ns/op
func BenchmarkLock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mtx := NewLock("lock1")
		err := mtx.Lock(context.Background())
		if err != nil {
			b.Fatal(err)
		}
		mtx.Unlock(context.Background())
	}
}

func TestLockMulti(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mtx := NewLock("test" + strconv.Itoa(i))
			err := mtx.Lock(ctx)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("lock success")
			mtx.Unlock(ctx)
		}()
	}
	wg.Wait()
}

func TestLockMulti2(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mtx := NewLock("test" + "1")
			err := mtx.Lock(ctx)
			require.NoError(t, err)

			// t.Log("lock success")
			mtx.Unlock(ctx)
		}()
	}
	wg.Wait()
}
