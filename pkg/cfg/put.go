package cfg

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"time"
)

func Put(addr string, version string, v string) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{addr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		zap.S().Warnf("连接 Etcd 失败: %v", err)
		return
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = cli.Put(ctx, configPath(version), v)
	if err != nil {
		zap.L().Error("put config err", zap.Error(err))
		return
	}
	zap.L().Info("put config success", zap.Any("config", v))
}
