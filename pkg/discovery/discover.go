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
	services map[string]*nodeGroup
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
		services: make(map[string]*nodeGroup),
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
		s.watch(cli)
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(time.Second * 2):
		}
	}
}

func (s *Discoverer) watch(cli *clientv3.Client) {
	revision := s.syncFullState(cli) // 重新全量拉取一次兜底
	if revision == 0 {
		// 全量拉取失败，等待 2 秒后重试
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(time.Second * 2):
		}
		return
	}
	rch := cli.Watch(s.ctx, s.prefix, clientv3.WithPrefix(), clientv3.WithRev(revision+1))
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
}

func (s *Discoverer) syncLoadLoop() {
	for {
		s.syncLoad()
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(time.Second * 2):
		}
	}
}

func (s *Discoverer) syncLoad() {
	pubsub := s.redisCli.Subscribe(s.ctx, RedisLoadChannel)
	defer pubsub.Close()

	// 确认是否订阅成功
	_, err := pubsub.Receive(s.ctx)
	if err != nil {
		zap.L().Error("failed to subscribe to redis channel", zap.Error(err))
		return
	}
	ch := pubsub.Channel()

	for {
		select {
		case <-s.ctx.Done():
			zap.L().Info("redis pub/sub stopped (context canceled)")
			return
		case msg, ok := <-ch:
			if !ok {
				zap.L().Warn("redis pub/sub channel closed")
				return
			}

			var loadMsg Node
			if err := json.Unmarshal([]byte(msg.Payload), &loadMsg); err != nil {
				continue
			}

			s.mtx.RLock()
			group, exists := s.services[loadMsg.SvcName]
			s.mtx.RUnlock()

			if exists {
				group.UpdateLoad(loadMsg.NodeID, loadMsg.Load)
			}
		}
	}
}

// syncFullState 全量加載
func (s *Discoverer) syncFullState(cli *clientv3.Client) (ver int64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	resp, err := cli.Get(ctx, s.prefix, clientv3.WithPrefix())
	if err != nil {
		zap.L().Error("Get etcd failed:", zap.Error(err))
		return 0
	}
	ver = resp.Header.Revision

	newAll := make(map[string]*nodeGroup)
	var keys []string
	var keyToMeta = make(map[string]struct {
		SerName string
		SerID   int32
	})

	for _, kv := range resp.Kvs {
		meta := Node{}
		if err = json.Unmarshal(kv.Value, &meta); err != nil {
			zap.L().Error("Unmarshal meta failed in loadParams", zap.Error(err))
			continue
		}
		svcName, serID, err := parseServicePath(string(kv.Key))
		if err != nil {
			zap.L().Error("parse service path failed in loadParams", zap.Error(err))
			continue
		}
		meta.NodeID = serID

		if newAll[svcName] == nil {
			newAll[svcName] = newNodeGroup()
		}
		newAll[svcName].Add(meta)

		key := redisKeyOfUpload(svcName, meta.NodeID)
		keys = append(keys, key)
		keyToMeta[key] = struct {
			SerName string
			SerID   int32
		}{SerName: svcName, SerID: meta.NodeID}
	}
	if len(keys) == 0 {
		return
	}
	// 使用 MGet 批量获取
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
		newAll[meta.SerName].UpdateLoad(meta.SerID, int32(load))
	}

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
	meta := Node{}
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
		one = newNodeGroup()
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
	one := s.services[svcName]
	defer s.mtx.Unlock()

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
