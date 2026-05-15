package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

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
}

func NewRegister(cli *clientv3.Client, redisCli redis.UniversalClient, svcName string, m *NodeMeta, ttl int64) (*Register, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Register{
		cli:      cli,
		redisCli: redisCli,
		key:      fmt.Sprintf("%s%s_%d", Prefix, svcName, m.NodeID),
		value:    string(b),
		ttl:      ttl,
		svcName:  svcName,
		svcID:    m.NodeID,
		ctx:      ctx,
		cancel:   cancel,
	}
	r.start()
	zap.L().Info("[service discover]service registered", zap.String("name", svcName), zap.Int32("id", m.NodeID))
	return r, nil
}

// register 启动注册和保活循环
func (s *Register) start() {
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
			}

			err := s.keepAliveLoop()
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
	}()

	// 启动 Redis 负载上报循环
	go s.reportLoadLoop()
}

func (s *Register) keepAliveLoop() error {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second*5)
	resp, err := s.cli.Grant(ctx, s.ttl)
	cancel()
	if err != nil {
		return err
	}
	s.leaseID.Store(int64(resp.ID))

	ctx, cancel = context.WithTimeout(s.ctx, time.Second*5)
	_, err = s.cli.Put(ctx, s.key, s.value, clientv3.WithLease(resp.ID))
	cancel()
	if err != nil {
		return err
	}

	keepAliveCh, err := s.cli.KeepAlive(s.ctx, resp.ID)
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

	redisKey := redisKeyOfUpload(s.svcName, s.svcID)

	for {
		select {
		case <-s.ctx.Done():
			// 退出时清理 Redis 中的负载数据
			s.redisCli.Del(context.Background(), redisKey)
			return
		case <-ticker.C:
			err := s.redisCli.Set(s.ctx, redisKey, s.currentLoad.Load(), time.Minute).Err()
			if err != nil {
				zap.L().Error("failed to report load to redis", zap.Error(err))
			} else {
				zap.L().Debug("[service discover] report load to redis", zap.String("server", s.svcName), zap.Int32("id", s.svcID), zap.Int32("load", s.currentLoad.Load()))
			}
		}
	}
}

// Close 优雅退出
func (s *Register) Close() {
	zap.L().Debug("[service discover]context canceled")
	s.cancel() // 停止所有后台循环

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
