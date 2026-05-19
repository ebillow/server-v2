package discovery

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	Prefix           = "/services/"
	RedisLoadChannel = "service:load:channel"
)

type Node struct {
	SvcName string `json:"svc_name"`
	NodeID  int32  `json:"node_id"`
	Load    int32  `json:"load"`
}

func redisKeyOfUpload(serName string, serID int32) string {
	return fmt.Sprintf("{server}:load:%s:%d", serName, serID)
}

var (
	register  *Register
	discovery *Discoverer
	etcdCli   *clientv3.Client
	redisCli  redis.UniversalClient
)

func Init(endpoints []string, rdb redis.UniversalClient) error {
	var err error
	etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		zap.L().Error("create etcd service failed", zap.Error(err))
		return err
	}
	redisCli = rdb

	return nil
}

// RegisterDefault 注册当前服务节点
func RegisterDefault(srvName string, m *Node) (err error) {
	// 初始化注册器：传入 etcd, redis 以及基础服务信息
	register, err = NewRegister(etcdCli, redisCli, srvName, m, 30)
	return err
}

func Watch() {
	if discovery == nil {
		discovery = NewDiscovery(etcdCli, redisCli, Prefix)
	}
}

// UpdateLoad 核心接口：业务层定时调用此方法上报当前进程负载
// 例如：UpdateLoad(int32(onlinePlayerCount))
func UpdateLoad(load int32) {
	if register != nil {
		register.UpdateLoad(load)
	}
}

func Close() {
	if register != nil {
		register.Close()
		time.Sleep(time.Millisecond * 50)
	}
	if etcdCli != nil {
		etcdCli.Close()
	}
}

func Exists(srvName string, id int32) bool {
	if discovery == nil {
		return false
	}
	return discovery.Exists(srvName, id)
}

func Select(srvName string) (id int32, ok bool) {
	if discovery == nil {
		return 0, false
	}
	return discovery.Select(srvName)
}
