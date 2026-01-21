package lock

import (
	"context"
	"server/modules/db"
	"sync"
	"testing"
	"time"
)

func TestSimpleLock(t *testing.T) {
	ctx := context.Background()
	t.Log("start lock 1")
	mtx := NewSimpleLock(db.Redis, "lock1", time.Second*8)
	ok, err := mtx.Lock(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	if !ok {
		t.Error("lock 1 fail")
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		t.Log("start lock 2")
		mtx := NewSimpleLock(db.Redis, "lock1", time.Second*8)
		ok, err := mtx.Lock(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		if !ok {
			t.Log("lock  2 fail")
			return
		}
	}()

	t.Log("do 1")
	time.Sleep(time.Second * 3)
	err = mtx.Unlock(ctx)
	if err != nil {
		t.Error(err)
	}
	t.Log("unlock 1")

	go func() {
		defer wg.Done()
		t.Log("start lock 3")
		mtx := NewSimpleLock(db.Redis, "lock1", time.Second*8)
		ok, err := mtx.Lock(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		if !ok {
			t.Error("lock 3 fail")
			return
		}
		time.Sleep(time.Second * 3)
		t.Log("do 3")
		err = mtx.Unlock(ctx)
		if err != nil {
			t.Error(err)
		}
		t.Log("unlock 3")
	}()
	wg.Wait()
}

// BenchmarkSimpleLock-10    	    2552	    425163 ns/op
func BenchmarkSimpleLock(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		mtx := NewSimpleLock(db.Redis, "lock1", time.Second*8)
		_, err := mtx.Lock(ctx)
		if err != nil {
			b.Error(err)
		}
		mtx.Unlock(ctx)
	}
}
