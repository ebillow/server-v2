package cfg

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"server/pkg/thread"
	"time"

	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote" // 必须导入 remote 包以支持 etcd
	clientv3 "go.etcd.io/etcd/client/v3"
)

func getVersion(name string, addr string) string {
	// 创建一个独立的 etcd 客户端用于 Watch
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{addr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		zap.S().Warnf("连接 Etcd 失败: %v", err)
		return ""
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	resp, err := cli.Get(ctx, fmt.Sprintf("/%s/version", name))
	if err != nil {
		zap.L().Warn("get version err", zap.Error(err))
		return ""
	}
	if len(resp.Kvs) == 0 {
		return "0.0"
	}
	return string(resp.Kvs[0].Value)
}

func Load(addr string, name string) {
	version := getVersion(name, addr)
	path := configPath(name, version)
	// 1. 初始化 Viper 基础配置
	err := viper.AddRemoteProvider("etcd3", addr, path)
	if err != nil {
		zap.L().Error("add remote provider failed")
	}
	viper.SetConfigType("yaml")

	// 2. 第一次读取配置
	if err := viper.ReadRemoteConfig(); err != nil {
		panic(err)
	}

	// 3. 将配置映射到结构体（初始加载）
	c := &Config{}
	if err := viper.Unmarshal(c); err != nil {
		panic(err)
	}
	cfg.Store(c)
	zap.S().Infof("初始配置: %+v\n", c)

	// 4. 启动一个 Goroutine 专门监听 Etcd 变化
	thread.GoSafe(func() {
		watch(addr, path)
	})
}

// watch 使用 etcd 原生客户端监听变化
func watch(endpoint, key string) {
	// 创建一个独立的 etcd 客户端用于 Watch
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		zap.S().Warnf("连接 Etcd 失败: %v", err)
		return
	}
	defer cli.Close()

	zap.S().Info("开始监听 Etcd 配置变化...")

	// Watch 返回一个通道，当 key 变化时会有数据
	rch := cli.Watch(context.Background(), key)

	for wresp := range rch {
		for _, ev := range wresp.Events {
			// 只有是 PUT (修改/新增) 操作时才处理
			if ev.Type == clientv3.EventTypePut {
				zap.S().Infof("配置修改: %s, 重新加载...\n", string(ev.Kv.Key))

				// 重新从 Remote 读取最新配置到 Viper
				// 注意：这里 ReadRemoteConfig 会去拉取最新的值更新 Viper 内部状态
				if err := viper.ReadRemoteConfig(); err != nil {
					zap.S().Warnf("Viper 重新读取配置失败: %v", err)
					continue
				}

				// 重新 Unmarshal 到结构体
				c := &Config{}
				if err := viper.Unmarshal(c); err != nil {
					zap.S().Warnf("Unmarshal 失败: %v", err)
					continue
				}

				cfg.Store(c)
				zap.S().Infof("配置更新成功: %+v\n", c)
			}
		}
	}
}
