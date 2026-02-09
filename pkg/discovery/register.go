package discovery

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type SDRegister struct {
	cli     *clientv3.Client
	leaseID clientv3.LeaseID
	key     string
	value   string
	ttl     int64
}

func newRegister(cli *clientv3.Client, serviceKey string, serviceVal string, ttl int64) *SDRegister {
	sr := &SDRegister{
		cli:   cli,
		key:   serviceKey,
		value: serviceVal,
		ttl:   ttl,
	}
	return sr
}

// Register 注册服务
func (s *SDRegister) register() error {
	ctx := context.TODO()

	// 1. 创建租约
	resp, err := s.cli.Grant(ctx, s.ttl)
	if err != nil {
		return err
	}
	s.leaseID = resp.ID

	// 2. 写入 KV 并绑定租约
	_, err = s.cli.Put(ctx, s.key, s.value, clientv3.WithLease(s.leaseID))
	if err != nil {
		return err
	}

	// 3. 永久续租
	keepAliveCh, err := s.cli.KeepAlive(ctx, s.leaseID)
	if err != nil {
		return err
	}

	// 4. 异步监听续租应答 (处理异常情况)
	go func() {
		for {
			select {
			case _, ok := <-keepAliveCh:
				if !ok {
					zap.L().Error("Lease keepalive channel closed, revoking...")
					// 生产环境策略：通常这里需要触发重试逻辑或重启服务
					// s.revoke()
					return
				}
				// 可以在这里记录 debug 日志：logger.Printf("Lease renewed: %d", ka.TTL)
			}
		}
	}()

	return nil
}

// Close 优雅退出
func (s *SDRegister) close() {
	// 撤销租约，立即从 etcd 删除节点
	_, err := s.cli.Revoke(context.Background(), s.leaseID)
	if err != nil {
		zap.L().Error("Failed to revoke lease", zap.Error(err))
	}
}
