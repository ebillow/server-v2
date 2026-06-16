package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"server/pkg/thread"
	"strconv"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Discoverer struct {
	services map[string]*nodeGroup
	mtx      sync.RWMutex

	watching map[string]struct{} // 记录已经处于监听中的服务
	watchMtx sync.Mutex          // 保护 watching map

	prefix   string
	redisCli redis.UniversalClient
	etcdCli  *clientv3.Client
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewDiscovery(cli *clientv3.Client, redisCli redis.UniversalClient, prefix string) *Discoverer {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Discoverer{
		prefix:   prefix,
		redisCli: redisCli,
		etcdCli:  cli,
		services: make(map[string]*nodeGroup),
		watching: make(map[string]struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
	return s
}

func (s *Discoverer) Close() {
	s.cancel()
	s.wg.Wait()
}

func (s *Discoverer) Watch(svcName string) {
	s.watchMtx.Lock()
	if _, ok := s.watching[svcName]; ok {
		s.watchMtx.Unlock()
		return // 已经处于监听状态，直接返回
	}
	s.watching[svcName] = struct{}{}
	s.watchMtx.Unlock()

	zap.L().Info("[service discover] start watching dependencies", zap.String("service", svcName))

	s.syncSvcFullState(svcName)

	s.wg.Add(2)
	thread.GoSafe(func() {
		defer s.wg.Done()
		s.watchEtcdLoop(svcName)
	})

	thread.GoSafe(func() {
		defer s.wg.Done()
		s.watchRedisLoop(svcName)
	})
}

// watchEtcdLoop 针对单一服务的断线重连监听机制
func (s *Discoverer) watchEtcdLoop(svcName string) {
	svcPrefix := etcdSvcPrefix(s.prefix, svcName)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		revision := s.syncSvcFullState(svcName)
		if revision == 0 {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Second * 2):
			}
			continue
		}

		rch := s.etcdCli.Watch(s.ctx, svcPrefix, clientv3.WithPrefix(), clientv3.WithRev(revision+1), clientv3.WithProgressNotify())
		for wresp := range rch {
			if wresp.Canceled {
				if err := wresp.Err(); err != nil {
					zap.L().Warn("etcd watch canceled",
						zap.Error(err),
						zap.Int64("compact_revision", wresp.CompactRevision),
					)
				}
				break
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
}

// syncSvcFullState 只针对特定服务做全量状态加载
func (s *Discoverer) syncSvcFullState(svcName string) (ver int64) {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second*3)
	defer cancel()

	svcPrefix := etcdSvcPrefix(s.prefix, svcName)
	resp, err := s.etcdCli.Get(ctx, svcPrefix, clientv3.WithPrefix())
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0
		}
		zap.L().Error("Get etcd failed:", zap.Error(err), zap.String("svc", svcName))
		return 0
	}

	newGroup := newNodeGroup()
	var keys []string
	var nodeIDs []int32

	for _, kv := range resp.Kvs {
		meta := Node{}
		if err = json.Unmarshal(kv.Value, &meta); err != nil {
			continue
		}
		_, serID, err := parseServicePath(string(kv.Key))
		if err != nil {
			continue
		}
		meta.NodeID = serID
		newGroup.Add(meta)

		keys = append(keys, redisKeyOfUpload(svcName, meta.NodeID))
		nodeIDs = append(nodeIDs, meta.NodeID)
	}

	// 拉取该组节点最新的 Redis Load
	if len(keys) > 0 {
		ctxM, cancelM := context.WithTimeout(s.ctx, time.Second*3)
		res, err := s.redisCli.MGet(ctxM, keys...).Result()
		cancelM()
		if err == nil {
			for i, val := range res {
				if val == nil {
					continue
				}
				if loadStr, ok := val.(string); ok {
					load, _ := strconv.Atoi(loadStr)
					newGroup.UpdateLoad(nodeIDs[i], int32(load))
				}
			}
		}
	}

	// 更新内存
	s.mtx.Lock()
	s.services[svcName] = newGroup
	s.mtx.Unlock()

	return resp.Header.Revision
}

func (s *Discoverer) watchRedisLoop(svcName string) {
	for {
		s.watchRedis(svcName)
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(time.Second*2 + time.Duration(rand.Intn(1000))*time.Millisecond):
		}
	}
}

func (s *Discoverer) watchRedis(svcName string) {
	channel := redisLoadChannel(svcName)
	pubsub := s.redisCli.Subscribe(s.ctx, channel)
	defer pubsub.Close()

	// 确认是否订阅成功
	_, err := pubsub.Receive(s.ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
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
			if err := sonic.Unmarshal([]byte(msg.Payload), &loadMsg); err != nil {
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

	one := s.services[svcName]
	if one == nil {
		one = newNodeGroup()
		s.services[svcName] = one
	}
	s.mtx.Unlock()

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

	if one != nil {
		if one.Delete(svcID) {
			delete(s.services, svcName)
		}
	}
	s.mtx.Unlock()

	zap.L().Info("[service discover]delete", zap.String("service", svcName), zap.Int32("id", svcID))
}

func (s *Discoverer) getService(svcName string) (*nodeGroup, bool) {
	s.mtx.RLock()
	one, ok := s.services[svcName]
	s.mtx.RUnlock()

	// 如果不存在，尝试触发懒加载
	if !ok {
		s.Watch(svcName)

		s.mtx.RLock()
		one, ok = s.services[svcName]
		s.mtx.RUnlock()

		return one, ok
	}

	return one, ok
}

// Select 基于最小负载,选择节点
func (s *Discoverer) Select(svcName string) (int32, bool) {
	one, ok := s.getService(svcName)
	if !ok {
		return 0, false
	}
	return one.SelectNode()
}

func (s *Discoverer) Exists(svcName string, id int32) bool {
	one, ok := s.getService(svcName)
	if !ok {
		return false
	}

	return one.Exists(id)
}

func (s *Discoverer) GetAllNodes(svcName string) ([]int32, bool) {
	one, ok := s.getService(svcName)
	if !ok {
		return nil, false
	}

	return one.AllNodeIDs(), true
}
