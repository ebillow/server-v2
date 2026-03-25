package discovery

import (
	"encoding/json"
	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"server/pkg/flag"
	"server/pkg/pb"
	"time"
)

const Prefix = "/services/"

/*
etcd 是基于 Raft 协议的一致性 KV 存储，
它的强项在于配置同步和节点发现，不擅长高频写操作。
如果你有 100 个服务节点，每秒更新一次负载，etcd 的磁盘 I/O 和网络同步会吃不消。
*/
type Meta struct {
	SerID int32
	Load  int32
}

var (
	register  *SDRegister
	discovery *Discovery
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

// Register 注册当前服务节点
func Register(serType pb.Server, serID int32) error {
	serName := flag.SrvName(serType)
	meta := &Meta{
		SerID: serID,
		Load:  0, // 初始负载为0
	}

	b, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	// 初始化注册器：传入 etcd, redis 以及基础服务信息
	register = newRegister(etcdCli, redisCli, serName, serID, string(b), 30)
	register.register()

	zap.L().Info("[service discover]service registered", zap.String("name", serName), zap.Int32("id", serID))
	return nil
}

func Watch() {
	if discovery == nil {
		discovery = newDiscovery(etcdCli, redisCli, Prefix)
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
		register.close()
		time.Sleep(time.Millisecond * 50)
	}
	if etcdCli != nil {
		etcdCli.Close()
	}
}

func Exist(serName string, id int32) bool {
	if discovery == nil {
		return false
	}
	return discovery.exist(serName, id)
}

func Pick(serName string) (id int32, ok bool) {
	if discovery == nil {
		return 0, false
	}
	return discovery.Pick(serName)
}
