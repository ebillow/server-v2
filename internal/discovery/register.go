package discovery

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

// 保存服务的权威元数据,在线人数等由redis处理

// Registrar 管理服务注册
type Registrar struct {
	client  *clientv3.Client
	leaseID clientv3.LeaseID
	key     string
	value   string
	ttl     int64
	cancel  context.CancelFunc
}

// NewRegistrar 创建一个注册器
func NewRegistrar(endpoints []string, key, value string, ttl int64) (*Registrar, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &Registrar{client: cli, key: key, value: value, ttl: ttl}, nil
}

// Register 注册服务并启动心跳
func (r *Registrar) Register() error {
	// 1. 申请租约
	resp, err := r.client.Grant(context.Background(), r.ttl)
	if err != nil {
		return err
	}
	r.leaseID = resp.ID

	// 2. Put 键值对并绑定租约
	if _, err := r.client.Put(context.Background(), r.key, r.value, clientv3.WithLease(r.leaseID)); err != nil {
		return err
	}

	// 3. 保活租约
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	ch, err := r.client.KeepAlive(ctx, r.leaseID)
	if err != nil {
		return err
	}

	// 4. 异步接收心跳响应
	go func() {
		for ka := range ch {
			if ka == nil {
				return // 租约失效
			}
		}
	}()
	return nil
}

// Deregister 取消注册
func (r *Registrar) Deregister() error {
	// 停止心跳
	if r.cancel != nil {
		r.cancel()
	}
	// 撤销租约，删除键值
	_, err := r.client.Revoke(context.Background(), r.leaseID)
	return err
}
