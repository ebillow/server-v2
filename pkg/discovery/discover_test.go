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

func registrar() *ServiceRegister {
	endpoints := []string{"127.0.0.1:2379"}
	serviceKey := "/services/myapp/"
	serviceVal := "127.0.0.1:8080"
	// 1. 服务注册
	rg, err := NewRegister(endpoints, serviceKey, serviceVal, 5)
	if err != nil {
		panic(err)
	}
	if err = rg.Register(); err != nil {
		panic(err)
	}
	fmt.Println("服务已注册：", serviceKey)
	return rg
}

func watch() *ServiceDiscovery {
	endpoints := []string{"127.0.0.1:2379"}
	watcher, err := NewServiceDiscovery(endpoints, "/services/")
	if err != nil {
		panic(err)
	}
	return watcher
}
func TestNewWatcher(t *testing.T) {
	watch()

	for i := 0; i < 10; i++ {
		r := registrar()
		time.Sleep(time.Second * 1)
		err := r.Register()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewWatcherAfterRegistrar(t *testing.T) {
	go func() {
		for i := 0; i < 10; i++ {
			r := registrar()
			time.Sleep(time.Second * 1)
			err := r.Register()
			if err != nil {
				t.Fatal(err)
			}
		}
	}()
	time.Sleep(time.Second * 1)
	watch()
	select {
	case <-time.After(time.Second * 10):
	}
}
