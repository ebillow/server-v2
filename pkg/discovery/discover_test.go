package discovery

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

func init() {
	// 初始化简单的 zap logger 用于测试输出
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

// setupClients 辅助方法：初始化底层的 Etcd 和 Redis 客户端
func setupClients(t *testing.T) (*clientv3.Client, redis.UniversalClient) {
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to connect etcd: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("failed to connect redis: %v", err)
	}

	// 每次测试前清理环境，防止脏数据影响
	_, _ = etcdCli.Delete(context.Background(), "/service/test_prod/", clientv3.WithPrefix())
	rdb.FlushDB(context.Background())

	return etcdCli, rdb
}

// 场景 1：测试懒加载触发机制
func TestLazyLoad(t *testing.T) {
	etcdCli, rdb := setupClients(t)
	defer etcdCli.Close()
	defer rdb.Close()

	// 1. 初始化 Manager
	mgr, err := NewManager("test_prod", []string{"127.0.0.1:2379"}, rdb)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	defer mgr.Close()

	// 2. 模拟下游服务注册：启动一个名叫 "chat_service" 的节点，ID=101
	node := &Node{
		SvcName: "chat_service",
		NodeID:  101,
		Load:    0,
	}
	err = mgr.Register("chat_service", node)
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	// 3. 消费者 Manager 初始化发现机制（注意：此时不预加载任何服务）
	mConsumer, _ := NewManager("test_prod", []string{"127.0.0.1:2379"}, rdb)
	mConsumer.Watch() // 空依赖
	defer mConsumer.Close()

	// 断言：此时 Consumer 内部没有 "chat_service" 的缓存
	mConsumer.discovery.mtx.RLock()
	_, exists := mConsumer.discovery.services["chat_service"]
	mConsumer.discovery.mtx.RUnlock()
	if exists {
		t.Fatal("expected service map to be empty before lazy load")
	}

	// 4. 触发懒加载：尝试 Select 节点
	// 懒加载会在内部发起同步 Etcd Get
	nodeID, ok := mConsumer.Select("chat_service")
	if !ok {
		t.Fatal("expected to find node after lazy loading, got none")
	}
	if nodeID != 101 {
		t.Fatalf("expected nodeID 101, got %d", nodeID)
	}
	t.Log("Lazy load triggered successfully!")
}

// 场景 2：测试负载更新与 Redis Pub/Sub 隔离
func TestLoadUpdateAndIsolation(t *testing.T) {
	etcdCli, rdb := setupClients(t)
	defer etcdCli.Close()
	defer rdb.Close()

	// 1. 初始化两套服务
	mChat, _ := NewManager("test_prod", []string{"127.0.0.1:2379"}, rdb)
	defer mChat.Close()
	mChat.Register("chat_service", &Node{SvcName: "chat_service", NodeID: 1, Load: 10})

	mCombat, _ := NewManager("test_prod", []string{"127.0.0.1:2379"}, rdb)
	defer mCombat.Close()
	mCombat.Register("combat_service", &Node{SvcName: "combat_service", NodeID: 2, Load: 100})

	// 2. 启动网关服务发现（只预加载 chat_service，不关心 combat_service）
	mGateway, _ := NewManager("test_prod", []string{"127.0.0.1:2379"}, rdb)
	mGateway.Watch("chat_service")
	defer mGateway.Close()

	// 给一定时间让全量拉取和 Watch 建立
	time.Sleep(1 * time.Second)

	// 3. 验证初始状态负载
	chatGrp := mGateway.discovery.services["chat_service"]
	load, ok := chatGrp.GetLoad(1)
	if !ok || load != 10 {
		t.Fatalf("expected chat_service node 1 load to be 10, got %v", load)
	}

	// 4. 动态更新 chat_service 的负载，触发 Redis Channel
	mChat.UpdateLoad(50)

	// 等待 reportLoadLoop 的 ticker（2秒） + Redis PubSub 处理时间
	time.Sleep(3 * time.Second)

	// 验证负载是否自动同步到网关服务
	newLoad, _ := chatGrp.GetLoad(1)
	if newLoad != 50 {
		t.Fatalf("expected chat_service load updated to 50, got %d", newLoad)
	}

	// 5. 验证隔离性：战斗服更新负载，网关毫不关心
	mCombat.UpdateLoad(999)
	time.Sleep(3 * time.Second)

	mGateway.discovery.mtx.RLock()
	_, combatExists := mGateway.discovery.services["combat_service"]
	mGateway.discovery.mtx.RUnlock()

	if combatExists {
		t.Fatal("gateway should not have loaded combat_service state")
	}
	t.Log("Redis isolation and load sync tested successfully!")
}

// 场景 3：测试 etcd 节点下线感知 (Watch)
func TestNodeOffline(t *testing.T) {
	etcdCli, rdb := setupClients(t)
	defer etcdCli.Close()
	defer rdb.Close()

	mSvc, _ := NewManager("test_prod", []string{"127.0.0.1:2379"}, rdb)
	mSvc.Register("auth_service", &Node{SvcName: "auth_service", NodeID: 88, Load: 0})

	mGateway, _ := NewManager("test_prod", []string{"127.0.0.1:2379"}, rdb)
	mGateway.Watch("auth_service")
	defer mGateway.Close()

	time.Sleep(1 * time.Second) // 等待建立监听

	// 1. 验证可找到该节点
	id, ok := mGateway.Select("auth_service")
	if !ok || id != 88 {
		t.Fatalf("failed to initial select auth_service")
	}

	// 2. 将服务强行关闭，触发 Etcd Revoke/Delete
	mSvc.Close()

	// 等待 etcd Watch Delete 事件传达到 Gateway
	time.Sleep(2 * time.Second)

	// 3. 再次获取，此时节点应已移除
	_, ok = mGateway.Select("auth_service")
	if ok {
		t.Fatal("expected auth_service to be offline and removed, but select succeeded")
	}
	t.Log("Node offline awareness tested successfully!")
}

// 场景 4：测试 P2C (两次随机选择) 负载均衡逻辑的准确性
func TestP2CSelection(t *testing.T) {
	// P2C 的核心在 NodeGroup 中，我们可以只测试内存里的 NodeGroup
	group := newNodeGroup()

	// 放入三个节点，负载各不相同
	group.Add(Node{NodeID: 1, Load: 100})
	group.Add(Node{NodeID: 2, Load: 50})
	group.Add(Node{NodeID: 3, Load: 10})

	// 测试 P2C 统计：10万次模拟调用，因为 Load=10 的最空闲，应该被选中次数最多
	counts := map[int32]int{1: 0, 2: 0, 3: 0}

	for i := 0; i < 100000; i++ {
		id, ok := group.SelectNode()
		if !ok {
			t.Fatal("SelectNode failed")
		}
		counts[id]++
	}

	fmt.Printf("P2C Distribution: Node1(Load:100): %d, Node2(Load:50): %d, Node3(Load:10): %d\n",
		counts[1], counts[2], counts[3])

	// 断言：由于 P2C 算法特性，负载最低的 Node 3 被选中的概率应该是最大的
	// 负载最高的 Node 1 应该最少
	if counts[3] <= counts[2] || counts[2] <= counts[1] {
		t.Fatalf("P2C balance seems incorrect, distribution: %v", counts)
	}
}
