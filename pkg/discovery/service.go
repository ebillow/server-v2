package discovery

import (
	"fmt"
	"server/pkg/gerror"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	RedisLoadChannel = "service:load:channel"
)

type Node struct {
	SvcName string `json:"svc_name"`
	NodeID  int32  `json:"node_id"`
	Load    int32  `json:"load"`
}

type Manager struct {
	register  *Register
	discovery *Discoverer
	etcdCli   *clientv3.Client
	redisCli  redis.UniversalClient
	prefix    string
}

var Default *Manager

func InitDefault(prod string, endpoints []string, rdb redis.UniversalClient) error {
	mgr, err := NewManager(prod, endpoints, rdb)
	if err != nil {
		return err
	}
	Default = mgr
	return nil
}

func NewManager(prod string, endpoints []string, rdb redis.UniversalClient) (*Manager, error) {
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:            endpoints,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    10 * time.Second,
		DialKeepAliveTimeout: 3 * time.Second,
		AutoSyncInterval:     time.Minute,
	})
	if err != nil {
		zap.L().Error("create etcd service failed", zap.Error(err))
		return nil, err
	}

	return &Manager{
		prefix:   fmt.Sprintf("/service/%s", prod),
		etcdCli:  etcdCli,
		redisCli: rdb,
	}, nil
}

// Register 注册当前服务节点
func (m *Manager) Register(srvName string, n *Node) (err error) {
	m.register, err = NewRegister(m.etcdCli, m.redisCli, m.prefix, srvName, n, 30)
	return err
}

func (m *Manager) Watch() {
	m.discovery = NewDiscovery(m.etcdCli, m.redisCli, m.prefix)
}

// UpdateLoad 核心接口：业务层定时调用此方法上报当前进程负载
// 例如：UpdateLoad(int32(onlinePlayerCount))
func (m *Manager) UpdateLoad(load int32) {
	if m.register != nil {
		m.register.UpdateLoad(load)
	}
}

func (m *Manager) Close() {
	if m.register != nil {
		m.register.Close()
	}
	if m.discovery != nil {
		m.discovery.Close()
	}
	if m.etcdCli != nil {
		m.etcdCli.Close()
	}
}

func (m *Manager) Exists(srvName string, id int32) bool {
	if m.discovery == nil {
		return false
	}
	return m.discovery.Exists(srvName, id)
}

func (m *Manager) Select(srvName string) (id int32, ok bool) {
	if m.discovery == nil {
		return 0, false
	}
	return m.discovery.Select(srvName)
}

func redisKeyOfUpload(svcName string, serID int32) string {
	return fmt.Sprintf("{server}:load:%s:%d", svcName, serID)
}
func etcdPath(prefix string, svcName string, serID int32) string {
	return fmt.Sprintf("%s/%s/%d", prefix, svcName, serID)
}

func parseServicePath(key string) (srvName string, serID int32, err error) {
	parts := strings.Split(strings.Trim(key, "/"), "/")

	if len(parts) < 2 {
		return "", 0, gerror.Newf("svc key is err:%s", key)
	}
	srvName = parts[len(parts)-2]
	idStr := parts[len(parts)-1]

	idx, err := strconv.Atoi(idStr)
	if err != nil {
		return "", 0, err
	}
	return srvName, int32(idx), nil
}
