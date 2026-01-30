package discovery

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"time"
)

type ServiceRegister struct {
	cli     *clientv3.Client
	leaseID clientv3.LeaseID
	key     string
	val     string
	ttl     int64
}

func NewRegister(endpoints []string, serviceKey string, serviceVal string, ttl int64) (*ServiceRegister, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	sr := &ServiceRegister{
		cli: cli,
		key: serviceKey,
		val: serviceVal,
		ttl: ttl,
	}
	return sr, nil
}

// Register 注册服务
func (s *ServiceRegister) Register() error {
	ctx := context.TODO()

	// 1. 创建租约
	resp, err := s.cli.Grant(ctx, s.ttl)
	if err != nil {
		return err
	}
	s.leaseID = resp.ID

	// 2. 写入 KV 并绑定租约
	_, err = s.cli.Put(ctx, s.key, s.val, clientv3.WithLease(s.leaseID))
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
					log.Println("Lease keepalive channel closed, revoking...")
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
func (s *ServiceRegister) Close() {
	// 撤销租约，立即从 etcd 删除节点
	s.cli.Revoke(context.Background(), s.leaseID)
}
