package discovery

import (
	"context"
	"encoding/json"
	"server/pkg/db"
	"server/pkg/logger"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.NewZapLog("../../bin/log/test.log", logger.Config{
		Level:   -1,
		Console: true,
	})
	err := db.InitRedis(db.RedisCfg{
		Addr: []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
	}, 0)
	if err != nil {
		panic(err)
	}
	err = Init([]string{"127.0.0.1:2379"}, db.Redis)
	if err != nil {
		panic(err)
	}
	m.Run()
}

// ============================================================================
// 测试 1：路径解析测试 (测试边界条件和异常输入)
// ============================================================================
func TestParseServicePath(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		wantSvc    string
		wantNodeID int32
		wantErr    bool
	}{
		{"正常路径", "/micro/registry/user_service_1001", "user_service", 1001, false},
		{"包含多个下划线", "/micro/registry/my_complex_svc_name_2002", "my_complex_svc_name", 2002, false},
		{"缺少ID", "/micro/registry/user_service", "", 0, true},
		{"ID不是数字", "/micro/registry/user_service_abc", "", 0, true},
		{"空字符串", "", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, id, err := parseServicePath(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSvc, svc)
				assert.Equal(t, tt.wantNodeID, id)
			}
		})
	}
}

func TestNodeGroup_CRUD(t *testing.T) {
	ng := newNodeGroup()

	// 1. Add
	ng.Add(Node{NodeID: 101, Load: 50})
	assert.True(t, ng.Exists(101))

	load, ok := ng.GetLoad(101)
	assert.True(t, ok)
	assert.Equal(t, int32(50), load)

	// 2. UpdateLoad (无锁更新)
	ng.UpdateLoad(101, 99)
	load, _ = ng.GetLoad(101)
	assert.Equal(t, int32(99), load)

	// 3. Delete
	isEmpty := ng.Delete(101)
	assert.True(t, isEmpty)
	assert.False(t, ng.Exists(101))
}

func TestNodeGroup_P2C_Algorithm(t *testing.T) {
	ng := newNodeGroup()

	// 模拟 3 个节点，负载差异巨大
	ng.Add(Node{NodeID: 1, Load: 10})  // 极低负载
	ng.Add(Node{NodeID: 2, Load: 50})  // 中等负载
	ng.Add(Node{NodeID: 3, Load: 200}) // 极高负载

	counts := make(map[int32]int)
	iterations := 100000 // 模拟十万次高并发请求

	for i := 0; i < iterations; i++ {
		id, ok := ng.SelectNode()
		assert.True(t, ok)
		counts[id]++
	}

	t.Logf("P2C 命中分布: Node1(Load:10)=%d, Node2(Load:50)=%d, Node3(Load:200)=%d",
		counts[1], counts[2], counts[3])

	// P2C 算法特性验证：
	// 1. 绝大部分流量应该打到节点 1
	assert.Greater(t, counts[1], 60000, "低负载节点应该获得大部分流量")
	// 2. 节点 2 会分担一部分流量，防止节点 1 被瞬间压垮
	assert.Greater(t, counts[2], 25000, "中等负载节点应该获得部分流量")
	// 3. 节点 3 作为高负载节点，应该被完美保护（流量接近于 0）
	assert.Equal(t, 0, counts[3], "极高负载节点在有其他选择时，应该被完全保护")
}

func TestNodeGroup_Concurrency(t *testing.T) {
	ng := newNodeGroup()
	ng.Add(Node{NodeID: 1, Load: 0})
	ng.Add(Node{NodeID: 2, Load: 0})

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// 1. 开启 10 个 Goroutine 疯狂进行无锁路由选择 (SelectNode)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					ng.SelectNode()
				}
			}
		}()
	}

	// 2. 开启 5 个 Goroutine 疯狂进行无锁负载更新 (UpdateLoad)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			load := int32(0)
			for {
				select {
				case <-stopCh:
					return
				default:
					load++
					ng.UpdateLoad(1, load)
					ng.UpdateLoad(2, load)
				}
			}
		}(i)
	}

	// 3. 开启 2 个 Goroutine 模拟节点动态上下线 (Add/Delete)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeID := int32(100 + id)
			for {
				select {
				case <-stopCh:
					return
				default:
					ng.Add(Node{NodeID: nodeID, Load: 0})
					time.Sleep(time.Millisecond * 10)
					ng.Delete(nodeID)
					time.Sleep(time.Millisecond * 10)
				}
			}
		}(i)
	}

	time.Sleep(time.Second * 2)
	close(stopCh)
	wg.Wait()

	// 如果没有 Panic，且 go test -race 不报错，说明无锁架构完美！
	t.Log("并发测试通过，未发生 Data Race 或 Panic")
}

// ============================================================================
// 测试 5：Redis Pub/Sub 负载同步测试 (结合 Miniredis)
// ============================================================================
func TestDiscoverer_RedisPubSub(t *testing.T) {
	// 1. 启动 mock redis server
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{mr.Addr()},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. 初始化 Discoverer
	d := &Discoverer{
		services: make(map[string]*nodeGroup),
		redisCli: rdb,
		ctx:      ctx,
		cancel:   cancel,
	}

	svcName := "login_service"
	nodeID := int32(888)

	// 手动塞入一个初始负载为 0 的节点
	d.services[svcName] = newNodeGroup()
	d.services[svcName].Add(Node{NodeID: nodeID, Load: 0})

	// 3. 在后台启动 Redis 订阅监听
	go d.syncLoad()

	// 给订阅一点启动时间
	time.Sleep(time.Millisecond * 100)

	// 4. 模拟 Register 端向 Redis 发送 Pub/Sub 广播
	msg := Node{
		SvcName: svcName,
		NodeID:  nodeID,
		Load:    150,
	}
	b, _ := json.Marshal(msg)
	err = rdb.Publish(ctx, RedisLoadChannel, string(b)).Err()
	assert.NoError(t, err)

	// 给 Discoverer 一点处理消息的时间
	time.Sleep(time.Millisecond * 100)

	// 5. 验证 Discoverer 内存中的负载是否被更新
	d.mtx.RLock()
	group := d.services[svcName]
	d.mtx.RUnlock()

	load, ok := group.GetLoad(nodeID)
	assert.True(t, ok)
	assert.Equal(t, int32(150), load, "Discoverer 应该通过 Pub/Sub 成功更新内存负载")
}

// func TestPick(t *testing.T) {
// 	Watch()
//
// 	r1, err := NewRegister(etcdCli, redisCli, flag.SrvName(pb.Server_Game), &Node{NodeID: 1}, 30)
// 	require.NoError(t, err)
// 	r1.UpdateLoad(2)
// 	time.Sleep(time.Second)
//
// 	exist := Exists(flag.SrvName(pb.Server_Game), 1)
// 	require.True(t, exist)
// 	exist = Exists(flag.SrvName(pb.Server_Game), 2)
// 	require.False(t, exist)
//
// 	r2, err := NewRegister(etcdCli, redisCli, flag.SrvName(pb.Server_Game), &Node{NodeID: 2}, 30)
// 	require.NoError(t, err)
// 	r2.UpdateLoad(4)
// 	time.Sleep(time.Second)
//
// 	exist = Exists(flag.SrvName(pb.Server_Game), 1)
// 	require.True(t, exist)
// 	exist = Exists(flag.SrvName(pb.Server_Game), 2)
// 	require.True(t, exist)
//
// 	time.Sleep(time.Second * 6)
//
// 	id, ok := Select(flag.SrvName(pb.Server_Game))
// 	require.True(t, ok)
// 	require.Equal(t, int32(1), id)
//
// 	r1.UpdateLoad(5)
// 	time.Sleep(time.Second * 6)
//
// 	id, ok = Select(flag.SrvName(pb.Server_Game))
// 	require.True(t, ok)
// 	require.Equal(t, int32(2), id)
//
// 	exist = Exists(flag.SrvName(pb.Server_Game), 1)
// 	require.True(t, exist)
// 	exist = Exists(flag.SrvName(pb.Server_Game), 2)
// 	require.True(t, exist)
//
// 	r1.Close()
//
// 	exist = Exists(flag.SrvName(pb.Server_Game), 1)
// 	require.False(t, exist)
// 	exist = Exists(flag.SrvName(pb.Server_Game), 2)
// 	require.True(t, exist)
//
// 	r2.Close()
//
// 	exist = Exists(flag.SrvName(pb.Server_Game), 1)
// 	require.False(t, exist)
// 	exist = Exists(flag.SrvName(pb.Server_Game), 2)
// 	require.False(t, exist)
//
// 	Close()
// }
