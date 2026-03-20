package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"math"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Discovery struct {
	all      map[string]*OneService
	mtx      sync.RWMutex
	prefix   string
	redisCli redis.UniversalClient
	ctx      context.Context
	cancel   context.CancelFunc
}

type OneService struct {
	mtx     sync.RWMutex
	serList map[int32]Meta
	// nodes   []Meta // 读写分离：缓存的切片，用于无锁/读锁的高频 pick
	minLoadSer int32
}

// Node 缓存结构重建
func (o *OneService) rebuildNodes() {
	// newNodes := make([]Meta, 0, len(o.serList))
	minLoad := int32(math.MaxInt32)
	for _, v := range o.serList {
		// newNodes = append(newNodes, v)
		if v.Load < minLoad {
			o.minLoadSer = v.SerID
		}
	}
	// o.nodes = newNodes
}

func (o *OneService) add(meta Meta) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	o.serList[meta.SerID] = meta
	o.rebuildNodes()
}

func (o *OneService) delete(serID int32) bool {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	delete(o.serList, serID)
	o.rebuildNodes()
	return len(o.serList) == 0
}

// updateLoad 更新特定节点的负载
func (o *OneService) updateLoad(serID int32, load int32) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	if meta, ok := o.serList[serID]; ok {
		meta.Load = load
		o.serList[serID] = meta
		o.rebuildNodes()
	}
}

// pickLeastLoad 基于本地缓存获取负载最小的节点
func (o *OneService) pickLeastLoad() (int32, bool) {
	o.mtx.RLock()
	defer o.mtx.RUnlock()

	// if len(o.nodes) == 0 {
	// 	return 0, false
	// }
	//
	// minNode := o.nodes[0]
	// for i := 1; i < len(o.nodes); i++ {
	// 	if o.nodes[i].Load < minNode.Load {
	// 		minNode = o.nodes[i]
	// 	}
	// }
	// return minNode.SerID, true
	if len(o.serList) == 0 {
		return 0, false
	}
	return o.minLoadSer, true
}

func newDiscovery(cli *clientv3.Client, redisCli redis.UniversalClient, prefix string) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Discovery{
		prefix:   prefix,
		redisCli: redisCli,
		all:      make(map[string]*OneService),
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
func (s *Discovery) watchLoop(cli *clientv3.Client) {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		s.loadParams(cli) // 重新全量拉取一次兜底

		rch := cli.Watch(s.ctx, s.prefix, clientv3.WithPrefix())
		for wresp := range rch {
			if wresp.Canceled {
				zap.L().Warn("etcd watch canceled, reconnecting...")
				break // 跳出内部循环，触发重连
			}
			for _, ev := range wresp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					s.setService(string(ev.Kv.Key), ev.Kv.Value)
				case clientv3.EventTypeDelete:
					s.delService(string(ev.Kv.Key))
				}
			}
		}
		// 重连前短暂休眠，避免雪崩
		time.Sleep(time.Second * 2)
	}
}

// syncLoadLoop 定时从 Redis 拉取最新负载数据
func (s *Discovery) syncLoadLoop() {
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mtx.RLock()
			// 收集当前需要同步的 service names
			var serNames []string
			for name := range s.all {
				serNames = append(serNames, name)
			}
			s.mtx.RUnlock()

			pipe := s.redisCli.Pipeline()
			for _, serName := range serNames {
				redisKey := fmt.Sprintf("server:load:%s", serName)
				pipe.HGetAll(s.ctx, redisKey)
			}
			cmds, err := pipe.Exec(s.ctx)
			if err != nil {
				zap.L().Error("failed to get load from redis", zap.Error(err))
				continue
			}

			for i, cmd := range cmds {
				res, err := cmd.(*redis.MapStringStringCmd).Result()
				serName := serNames[i]
				if err != nil {
					if !errors.Is(err, redis.Nil) {
						zap.L().Error("failed to get load from redis", zap.String("serName", serName), zap.Error(err))
					}
					continue
				}

				s.mtx.RLock()
				oneSrv, ok := s.all[serName]
				s.mtx.RUnlock()

				if ok {
					for idStr, loadStr := range res {
						serID, _ := strconv.Atoi(idStr)
						load, _ := strconv.Atoi(loadStr)
						oneSrv.updateLoad(int32(serID), int32(load))
					}
				}
			}
		}
	}
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
		s.setService(string(kv.Key), kv.Value)
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

func (s *Discovery) setService(key string, val []byte) {
	meta := Meta{}
	err := json.Unmarshal(val, &meta)
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
	zap.L().Info("[service discover]add", zap.String("service", serName), zap.Int32("id", serID), zap.Any("meta", meta))
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

	zap.L().Info("[service discover]delete", zap.String("service", serName), zap.Int32("id", serID))
}

// Pick 对外暴露的 Pick 接口，基于最小负载
func (s *Discovery) Pick(serName string) (int32, bool) {
	s.mtx.RLock()
	one, ok := s.all[serName]
	s.mtx.RUnlock()

	if !ok {
		return 0, false
	}
	return one.pickLeastLoad()
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
