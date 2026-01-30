package discovery

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type ServiceDiscovery struct {
	cli        *clientv3.Client
	serverList sync.Map // 使用 sync.Map 保證併發讀寫安全，或者使用 RWMutex
	prefix     string
	index      uint64 // 用於 RoundRobin 輪詢
}

func NewServiceDiscovery(endpoints []string, prefix string) (*ServiceDiscovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	s := &ServiceDiscovery{
		cli:    cli,
		prefix: prefix,
	}

	// 啟動時先加載一次
	s.loadParams()
	// 啟動監聽
	go s.watcher()

	return s, nil
}

// loadParams 全量加載 (用於初始化和兜底)
func (s *ServiceDiscovery) loadParams() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	resp, err := s.cli.Get(ctx, s.prefix, clientv3.WithPrefix())
	if err != nil {
		log.Println("Get etcd failed:", err)
		return
	}

	// 簡單的做法是清空重建，或者在這裡做 Diff
	// 這裡演示簡單邏輯：遍歷存入
	for _, kv := range resp.Kvs {
		s.SetService(string(kv.Key), string(kv.Value))
	}
}

// watcher 核心監聽邏輯
func (s *ServiceDiscovery) watcher() {
	rch := s.cli.Watch(context.Background(), s.prefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut: // 新增或修改
				log.Printf("Service update: %s : %s\n", ev.Kv.Key, ev.Kv.Value)
				s.SetService(string(ev.Kv.Key), string(ev.Kv.Value))
			case clientv3.EventTypeDelete: // 下線
				log.Printf("Service delete: %s\n", ev.Kv.Key)
				s.DelService(string(ev.Kv.Key))
			}
		}
	}
}

func (s *ServiceDiscovery) SetService(key, val string) {
	s.serverList.Store(key, val)
}

func (s *ServiceDiscovery) DelService(key string) {
	s.serverList.Delete(key)
}

// Pick 獲取服務地址 (實現 Round Robin 負載均衡)
func (s *ServiceDiscovery) Pick() string {
	var list []string
	s.serverList.Range(func(k, v interface{}) bool {
		list = append(list, v.(string))
		return true
	})

	if len(list) == 0 {
		return ""
	}

	// 原子操作實現輪詢
	next := atomic.AddUint64(&s.index, 1)
	return list[next%uint64(len(list))]
}
