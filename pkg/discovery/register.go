package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"server/pkg/thread"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type Register struct {
	cli         *clientv3.Client
	redisCli    redis.UniversalClient
	leaseID     atomic.Int64
	key         string
	value       string
	ttl         int64
	svcName     string
	svcID       int32
	ctx         context.Context
	cancel      context.CancelFunc
	currentLoad atomic.Int32 // 业务层更新此值
	wg          sync.WaitGroup
	first       bool
}

func NewRegister(cli *clientv3.Client, redisCli redis.UniversalClient, prefix string, svcName string, m *Node, ttl int64) (*Register, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Register{
		cli:      cli,
		redisCli: redisCli,
		key:      etcdPath(prefix, svcName, m.NodeID),
		value:    string(b),
		ttl:      ttl,
		svcName:  svcName,
		svcID:    m.NodeID,
		ctx:      ctx,
		cancel:   cancel,
		first:    true,
	}

	err = r.initialRegister()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("initial etcd register failed: %w", err)
	}

	r.wg.Add(2)
	thread.GoSafe(func() {
		defer r.wg.Done()
		r.keepAliveLoop()
	})
	thread.GoSafe(func() {
		defer r.wg.Done()
		r.reportLoadLoop()
	})

	zap.L().Info("[service discover]service registered", zap.String("name", svcName), zap.Int32("id", m.NodeID))
	return r, nil
}

func (s *Register) keepAliveLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		err := s.keepAlive()
		if err != nil {
			// 检查是否是由于 context 取消导致的错误
			// 或者是 context 已经 done 了
			select {
			case <-s.ctx.Done():
				// 说明是正常关闭，无需打印 Error 日志，直接退出
				zap.L().Info("[service discover]etcd keepalive stopped (context canceled)")
				return
			default:
				// 说明是真正的网络或 etcd 故障，打印错误并重试
				zap.L().Error("[service discover]etcd keepalive failed, retrying...", zap.Error(err))
			}
		}
		t := time.NewTimer(time.Second * 3)
		select {
		case <-t.C:
		case <-s.ctx.Done():
			t.Stop()
			return
		}
	}
}

func (s *Register) initialRegister() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := s.cli.Grant(ctx, s.ttl)
	if err != nil {
		return err
	}
	s.leaseID.Store(int64(resp.ID))

	_, err = s.cli.Put(ctx, s.key, s.value, clientv3.WithLease(resp.ID))
	return err
}

func (s *Register) keepAlive() error {
	if !s.first {
		_ = s.initialRegister()
		s.first = false
	}

	keepAliveCh, err := s.cli.KeepAlive(s.ctx, clientv3.LeaseID(s.leaseID.Load()))
	if err != nil {
		return err
	}

	for {
		select {
		case <-s.ctx.Done():
			return nil
		case _, ok := <-keepAliveCh:
			if !ok {
				return fmt.Errorf("keepalive channel closed")
			}
			// 正常续租
		}
	}
}

// UpdateLoad 暴露给业务层调用的方法（仅更新内存变量）
func (s *Register) UpdateLoad(load int32) {
	s.currentLoad.Store(load)
}

// reportLoadLoop 每隔一段时间将本地负载同步到 Redis
func (s *Register) reportLoadLoop() {
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()
	var lastReportedLoad int32
	var ticksSinceLastReport int

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			load := s.currentLoad.Load()
			ticksSinceLastReport++

			if math.Abs(float64(load-lastReportedLoad)) > 5 || ticksSinceLastReport > 5 {
				msg := Node{
					SvcName: s.svcName,
					NodeID:  s.svcID,
					Load:    load,
				}
				b, _ := sonic.MarshalString(msg)

				ctx, cancel := context.WithTimeout(s.ctx, time.Second)
				pipe := s.redisCli.Pipeline()
				pipe.Set(ctx, redisKeyOfUpload(s.svcName, s.svcID), load, time.Minute)
				pipe.Publish(ctx, redisLoadChannel(s.svcName), b)
				_, err := pipe.Exec(ctx)
				cancel()
				if err != nil {
					zap.L().Error("failed to report load to redis", zap.Error(err))
				}

				lastReportedLoad = load
				ticksSinceLastReport = 0
			}
		}
	}
}

// Close 优雅退出
func (s *Register) Close() {
	zap.L().Debug("[service discover]context canceled")
	s.cancel() // 停止所有后台循环
	s.wg.Wait()

	leaseID := s.leaseID.Load()
	if leaseID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		zap.L().Debug("[service discover]close revoke")
		_, err := s.cli.Revoke(ctx, clientv3.LeaseID(leaseID))
		if err != nil {
			zap.L().Error("Failed to revoke lease", zap.Error(err))
		}
	}
}
