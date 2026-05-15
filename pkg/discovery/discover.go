package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Discoverer struct {
	services map[string]*NodeGroup
	mtx      sync.RWMutex
	prefix   string
	redisCli redis.UniversalClient
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewDiscovery(cli *clientv3.Client, redisCli redis.UniversalClient, prefix string) *Discoverer {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Discoverer{
		prefix:   prefix,
		redisCli: redisCli,
		services: make(map[string]*NodeGroup),
		ctx:      ctx,
		cancel:   cancel,
	}

	// 启动带重试的 Watcher
	go s.watchLoop(cli)
	// 启动 Redis 负载同步循环
	go s.syncLoadLoop()

	return s
}

// watchLoop 包含断线重连的监听机制
func (s *Discoverer) watchLoop(cli *clientv3.Client) {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		revision := s.syncFullState(cli) // 重新全量拉取一次兜底

		rch := cli.Watch(s.ctx, s.prefix, clientv3.WithPrefix(), clientv3.WithRev(revision))
		for wresp := range rch {
			if wresp.Canceled {
				zap.L().Warn("etcd watch canceled, reconnecting...")
				break // 跳出内部循环，触发重连
			}
			for _, ev := range wresp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					s.upsertNode(string(ev.Kv.Key), ev.Kv.Value)
				case clientv3.EventTypeDelete:
					s.removeNode(string(ev.Kv.Key))
				}
			}
		}
		// 重连前短暂休眠，避免雪崩
		time.Sleep(time.Second * 2)
	}
}

// syncLoadLoop 定时从 Redis 拉取最新负载数据
func (s *Discoverer) syncLoadLoop() {
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.syncLoad()
		}
	}
}

func (s *Discoverer) syncLoad() {
	var keys []string
	var keyToMeta = make(map[string]struct {
		SerName string
		SerID   int32
	})

	s.mtx.RLock()
	groups := make(map[string]*NodeGroup, len(s.services))
	for svc, group := range s.services {
		groups[svc] = group
	}
	s.mtx.RUnlock()

	for svc, group := range groups {
		group.RangeReadOnly(func(m NodeMeta) {
			key := redisKeyOfUpload(svc, m.NodeID)
			keys = append(keys, key)
			keyToMeta[key] = struct {
				SerName string
				SerID   int32
			}{SerName: svc, SerID: m.NodeID}
		})
	}

	if len(keys) == 0 {
		return
	}

	res, err := s.redisCli.MGet(s.ctx, keys...).Result()
	if err != nil {
		zap.L().Error("failed to mget load from redis", zap.Error(err))
		return
	}

	for i, val := range res {
		if val == nil {
			continue // 节点可能刚启动还未上报
		}
		loadStr, ok := val.(string)
		if !ok {
			continue
		}
		load, _ := strconv.Atoi(loadStr)

		meta := keyToMeta[keys[i]]
		s.mtx.RLock()
		oneSrv, exists := s.services[meta.SerName]
		s.mtx.RUnlock()
		if exists {
			oneSrv.UpdateLoad(meta.SerID, int32(load))
		}
	}
}

// syncFullState 全量加載
func (s *Discoverer) syncFullState(cli *clientv3.Client) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	resp, err := cli.Get(ctx, s.prefix, clientv3.WithPrefix())
	if err != nil {
		zap.L().Error("Get etcd failed:", zap.Error(err))
		return 0
	}

	newAll := make(map[string]*NodeGroup)
	for _, kv := range resp.Kvs {
		meta := NodeMeta{}
		if err = json.Unmarshal(kv.Value, &meta); err != nil {
			zap.L().Error("Unmarshal meta failed in loadParams", zap.Error(err))
			continue
		}
		srvName, serID, err := parseServicePath(string(kv.Key))
		if err != nil {
			zap.L().Error("parse service path failed in loadParams", zap.Error(err))
			continue
		}
		meta.NodeID = serID

		if newAll[srvName] == nil {
			newAll[srvName] = NewNodeGroup()
		}
		newAll[srvName].Add(meta)
	}

	s.mtx.RLock() // 获取负载
	for serName, newOneSrv := range newAll {
		if oldOneSrv, exists := s.services[serName]; exists {
			// 此时 newOneSrv 还没暴露，不需要加
			for nodeID, newMeta := range newOneSrv.nodes {
				// 从"旧"的缓存中安全地获取历史负载
				if oldLoad, ok := oldOneSrv.GetLoad(nodeID); ok {
					newMeta.Load = oldLoad
					newOneSrv.nodes[nodeID] = newMeta
				}
			}
		}
		// 继承完 Load 后，统一重建一次 P2C 缓存切片
		newOneSrv.rebuildNodeCache()
	}
	s.mtx.RUnlock()

	s.mtx.Lock()
	s.services = newAll
	s.mtx.Unlock()

	zap.L().Info("[service discover] full replace completed", zap.Int("services_count", len(newAll)))
	return resp.Header.Revision
}

func parseServicePath(key string) (srvName string, serID int32, err error) {
	baseName := path.Base(key)
	lastIdx := strings.LastIndex(baseName, "_")
	if lastIdx == -1 {
		return "", 0, fmt.Errorf("invalid service path: %s", key)
	}

	srvName = baseName[:lastIdx]
	idStr := baseName[lastIdx+1:]

	idx, err := strconv.Atoi(idStr)
	if err != nil {
		return "", 0, err
	}
	return srvName, int32(idx), nil
}

func (s *Discoverer) upsertNode(key string, val []byte) {
	meta := NodeMeta{}
	err := json.Unmarshal(val, &meta)
	if err != nil {
		zap.L().Error("Unmarshal meta failed", zap.Error(err))
		return
	}
	svcName, svcID, err := parseServicePath(key)
	if err != nil {
		zap.L().Error("parse service path failed", zap.Error(err))
		return
	}
	meta.NodeID = svcID
	s.mtx.Lock()
	defer s.mtx.Unlock()
	one := s.services[svcName]
	if one == nil {
		one = NewNodeGroup()
		s.services[svcName] = one
	}
	one.Add(meta)
	zap.L().Info("[service discover]add", zap.String("service", svcName), zap.Int32("id", svcID), zap.Any("meta", meta))
}

func (s *Discoverer) removeNode(key string) {
	svcName, svcID, err := parseServicePath(key)
	if err != nil {
		zap.L().Error("parse service path failed", zap.Error(err))
		return
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	one := s.services[svcName]
	if one != nil {
		if one.Delete(svcID) {
			delete(s.services, svcName)
		}
	}

	zap.L().Info("[service discover]delete", zap.String("service", svcName), zap.Int32("id", svcID))
}

// Select 基于最小负载,选择节点
func (s *Discoverer) Select(svcName string) (int32, bool) {
	s.mtx.RLock()
	one, ok := s.services[svcName]
	s.mtx.RUnlock()

	if !ok {
		return 0, false
	}
	return one.SelectNode()
}

func (s *Discoverer) Exists(svcName string, id int32) bool {
	s.mtx.RLock()
	one, ok := s.services[svcName]
	s.mtx.RUnlock()

	if !ok {
		return false
	}

	return one.Exists(id)
}
