package discovery

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type SDRegister struct {
	cli         *clientv3.Client
	redisCli    redis.UniversalClient
	leaseID     clientv3.LeaseID
	key         string
	value       string
	ttl         int64
	serName     string
	serID       int32
	ctx         context.Context
	cancel      context.CancelFunc
	currentLoad atomic.Int32 // 业务层更新此值
}

func newRegister(cli *clientv3.Client, redisCli redis.UniversalClient, serName string, serID int32, value string, ttl int64) *SDRegister {
	ctx, cancel := context.WithCancel(context.Background())
	return &SDRegister{
		cli:      cli,
		redisCli: redisCli,
		key:      fmt.Sprintf("%s%s_%d", Prefix, serName, serID),
		value:    value,
		ttl:      ttl,
		serName:  serName,
		serID:    serID,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// register 启动注册和保活循环
func (s *SDRegister) register() {
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
			// 3. 重试间隔也要能被 context 中断
			select {
			case <-time.After(time.Second * 3):
			case <-s.ctx.Done():
				return
			}
		}
	}()

	// 启动 Redis 负载上报循环
	go s.reportLoadLoop()
}

func (s *SDRegister) keepAliveLoop() error {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second*5)
	resp, err := s.cli.Grant(ctx, s.ttl)
	cancel()
	if err != nil {
		return err
	}
	s.leaseID = resp.ID

	ctx, cancel = context.WithTimeout(s.ctx, time.Second*5)
	_, err = s.cli.Put(ctx, s.key, s.value, clientv3.WithLease(s.leaseID))
	cancel()
	if err != nil {
		return err
	}

	keepAliveCh, err := s.cli.KeepAlive(s.ctx, s.leaseID)
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
func (s *SDRegister) UpdateLoad(load int32) {
	s.currentLoad.Store(load)
}

// reportLoadLoop 每隔一段时间将本地负载同步到 Redis
func (s *SDRegister) reportLoadLoop() {
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	redisKey := fmt.Sprintf("server:load:%s", s.serName)
	fieldKey := fmt.Sprintf("%d", s.serID)

	for {
		select {
		case <-s.ctx.Done():
			// 退出时清理 Redis 中的负载数据
			s.redisCli.HDel(context.Background(), redisKey, fieldKey)
			return
		case <-ticker.C:
			pipe := s.redisCli.Pipeline()
			pipe.HSet(s.ctx, redisKey, fieldKey, s.currentLoad.Load())
			pipe.Expire(s.ctx, redisKey, time.Minute)
			_, err := pipe.Exec(s.ctx)
			if err != nil {
				zap.L().Error("failed to report load to redis", zap.Error(err))
			}
		}
	}
}

// close 优雅退出
func (s *SDRegister) close() {
	zap.L().Debug("[service discover]context canceled")
	s.cancel() // 停止所有后台循环
	if s.leaseID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		zap.L().Debug("[service discover]close revoke")
		_, err := s.cli.Revoke(ctx, s.leaseID)
		if err != nil {
			zap.L().Error("Failed to revoke lease", zap.Error(err))
		}
	}
}
