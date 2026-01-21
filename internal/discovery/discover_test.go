package discovery

import (
	"fmt"
	"testing"
	"time"
)

func TestNewRegistrar(t *testing.T) {
	registrar()
	select {
	case <-time.After(time.Minute):
		return
	}
}

func registrar() *Registrar {
	endpoints := []string{"127.0.0.1:2379"}
	serviceKey := "/services/myapp/"
	serviceVal := "127.0.0.1:8080"
	// 1. 服务注册
	rg, err := NewRegistrar(endpoints, serviceKey, serviceVal, 5)
	if err != nil {
		panic(err)
	}
	if err = rg.Register(); err != nil {
		panic(err)
	}
	fmt.Println("服务已注册：", serviceKey)
	return rg
}

func watch() {
	endpoints := []string{"127.0.0.1:2379"}
	watcher, err := NewWatcher(endpoints, "/services/",
		func(key, val string) { fmt.Println("实例上线：", key, val) },
		func(key string) { fmt.Println("实例下线：", key) },
	)
	if err != nil {
		panic(err)
	}
	if err = watcher.Watch(); err != nil {
		panic(err)
	}
}
func TestNewWatcher(t *testing.T) {
	watch()

	for i := 0; i < 10; i++ {
		r := registrar()
		time.Sleep(time.Second * 10)
		err := r.Deregister()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewWatcherAfterRegistrar(t *testing.T) {
	go func() {
		for i := 0; i < 10; i++ {
			r := registrar()
			time.Sleep(time.Second * 10)
			err := r.Deregister()
			if err != nil {
				t.Fatal(err)
			}
		}
	}()
	time.Sleep(time.Second * 10)
	watch()
	select {
	case <-time.After(time.Minute):
	}
}
