package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Discovery struct {
	all map[string]*OneService
	mtx sync.RWMutex

	prefix string
}

type OneService struct {
	serList map[int32]Meta
	index   uint64 // 用於 RoundRobin 輪詢
}

func (o *OneService) add(meta Meta) {
	o.serList[meta.SerID] = meta
}

func (o *OneService) delete(serID int32) bool {
	delete(o.serList, serID)
	if len(o.serList) == 0 {
		return true
	}
	return false
}

// pick 獲取服務地址 (實現 Round Robin 負載均衡)
func (o *OneService) pick() (int32, bool) {
	var list []int32
	for _, v := range o.serList {
		list = append(list, v.SerID)
	}

	if len(list) == 0 {
		return 0, false
	}

	// 原子操作實現輪詢
	next := atomic.AddUint64(&o.index, 1)
	return list[next%uint64(len(list))], true
}

func newDiscovery(cli *clientv3.Client, prefix string) *Discovery {
	s := &Discovery{
		prefix: prefix,
		all:    make(map[string]*OneService),
	}

	// 啟動時先加載一次
	s.loadParams(cli)
	// 啟動監聽
	go s.watcher(cli)

	return s
}

// loadParams 全量加載 (用於初始化和兜底)
func (s *Discovery) loadParams(cli *clientv3.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	resp, err := cli.Get(ctx, s.prefix, clientv3.WithPrefix())
	if err != nil {
		zap.L().Error("Get etcd failed:", zap.Error(err))
		return
	}

	// 簡單的做法是清空重建，或者在這裡做 Diff
	// 這裡演示簡單邏輯：遍歷存入
	for _, kv := range resp.Kvs {
		s.setService(string(kv.Key), string(kv.Value))
	}
}

// watcher 核心監聽邏輯
func (s *Discovery) watcher(cli *clientv3.Client) {
	rch := cli.Watch(context.Background(), s.prefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut: // 新增或修改
				// zap.S().Infof("Service update: %s : %s\n", ev.Kv.Key, ev.Kv.Value)
				s.setService(string(ev.Kv.Key), string(ev.Kv.Value))
			case clientv3.EventTypeDelete: // 下線
				// zap.S().Infof("Service delete: %s\n", ev.Kv.Key)
				s.delService(string(ev.Kv.Key))
			}
		}
	}
}

func parseServicePath(key string) (serName string, serID int32, err error) {
	baseName := path.Base(key)
	lastIdx := strings.LastIndex(baseName, "_")
	if lastIdx == -1 {
		return "", 0, fmt.Errorf("invalid service path: %s", key)
	}

	serName = baseName[:lastIdx]
	idStr := baseName[lastIdx+1:]

	idx, err := strconv.Atoi(idStr)
	if err != nil {
		return "", 0, err
	}
	return serName, int32(idx), nil
}

func (s *Discovery) setService(key, val string) {
	meta := Meta{}
	err := json.Unmarshal([]byte(val), &meta)
	if err != nil {
		zap.L().Error("Unmarshal meta failed", zap.Error(err))
		return
	}
	serName, serID, err := parseServicePath(key)
	if err != nil {
		zap.L().Error("parse service path failed", zap.Error(err))
		return
	}
	meta.SerID = serID
	s.mtx.Lock()
	defer s.mtx.Unlock()
	one := s.all[serName]
	if one == nil {
		one = &OneService{serList: make(map[int32]Meta)}
		s.all[serName] = one
	}
	one.add(meta)
	zap.L().Info("[service] add", zap.String("service", serName), zap.Int32("id", serID), zap.Any("meta", meta))
}

func (s *Discovery) delService(key string) {
	serName, serID, err := parseServicePath(key)
	if err != nil {
		zap.L().Error("parse service path failed", zap.Error(err))
		return
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	one := s.all[serName]
	if one != nil {
		if one.delete(serID) {
			delete(s.all, serName)
		}
	}

	zap.L().Info("[service] delete", zap.String("service", serName), zap.Int32("id", serID))
}

// pick 獲取服務地址 (實現 Round Robin 負載均衡)
func (s *Discovery) pick(serName string) (int32, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	one, ok := s.all[serName]
	if !ok {
		return 0, false
	}

	return one.pick()
}

func (s *Discovery) exist(serName string, id int32) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	one, ok := s.all[serName]
	if !ok {
		return false
	}
	_, ok = one.serList[id]
	return ok
}
