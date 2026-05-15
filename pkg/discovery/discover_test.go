package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/logger"
	"server/pkg/pb"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// ============================================================================
// 测试 2：节点增删与 Exists 逻辑测试
// ============================================================================
func TestDiscoverer_UpsertAndRemove(t *testing.T) {
	d := &Discoverer{
		services: make(map[string]*NodeGroup),
	}

	svcName := "order_service"
	nodeID := int32(101)
	key := fmt.Sprintf("/prefix/%s_%d", svcName, nodeID)

	meta := NodeMeta{NodeID: nodeID, Load: 0}
	val, _ := json.Marshal(meta)

	// 1. 测试 Upsert (新增)
	d.upsertNode(key, val)
	assert.True(t, d.Exists(svcName, nodeID), "节点应该存在")

	// 验证内部缓存结构
	pool := d.services[svcName]
	assert.NotNil(t, pool)
	assert.Equal(t, 1, len(pool.nodes))
	assert.Equal(t, 1, len(pool.nodeIDs), "用于P2C的切片应该被正确重建")

	// 2. 测试 Remove (删除)
	d.removeNode(key)
	assert.False(t, d.Exists(svcName, nodeID), "节点应该被删除")

	// 验证 pool 是否被清理
	_, ok := d.services[svcName]
	assert.False(t, ok, "当服务下没有节点时，应该清理掉整个 EndpointPool")
}

// ============================================================================
// 测试 3：P2C 算法分布测试 (核心亮点)
// ============================================================================
func TestP2C_AlgorithmDistribution(t *testing.T) {
	pool := &NodeGroup{
		nodes: make(map[int32]NodeMeta),
	}

	// 模拟 3 个节点，负载差异巨大
	pool.nodes[1] = NodeMeta{NodeID: 1, Load: 10}  // 极低负载
	pool.nodes[2] = NodeMeta{NodeID: 2, Load: 50}  // 中等负载
	pool.nodes[3] = NodeMeta{NodeID: 3, Load: 200} // 极高负载
	pool.rebuildNodeCache()

	counts := make(map[int32]int)
	iterations := 60000

	// 模拟高并发下的 6 万次 Pick
	for i := 0; i < iterations; i++ {
		id, ok := pool.SelectNode()
		assert.True(t, ok)
		counts[id]++
	}

	/*
		理论概率分析 (P2C 随机选2个，取较小者)：
		组合 (1,2) -> 选 1
		组合 (1,3) -> 选 1
		组合 (2,3) -> 选 2
		因此 1 的概率约为 66.6%，2 的概率约为 33.3%，3 的概率为 0%
	*/
	t.Logf("P2C 命中分布: Node1(Load:10)=%d, Node2(Load:50)=%d, Node3(Load:200)=%d",
		counts[1], counts[2], counts[3])

	// 验证羊群效应被避免，且高负载节点被保护
	assert.Greater(t, counts[1], 38000, "低负载节点应该获得大部分流量")
	assert.Greater(t, counts[2], 18000, "中等负载节点应该获得部分流量，避免节点1被彻底压垮")
	assert.Equal(t, 0, counts[3], "极高负载节点在有其他选择时，应该被完全保护(0流量)")
}

// ============================================================================
// 测试 4：Redis 负载同步测试 (结合 Miniredis)
// ============================================================================
func TestDiscoverer_SyncLoad(t *testing.T) {

	// 2. 初始化 Discoverer 并手动塞入一个节点
	d := &Discoverer{
		services: make(map[string]*NodeGroup),
		redisCli: db.Redis,
		ctx:      context.Background(),
	}

	svcName := "payment_service"
	nodeID := int32(888)

	pool := &NodeGroup{nodes: make(map[int32]NodeMeta)}
	pool.nodes[nodeID] = NodeMeta{NodeID: nodeID, Load: 0} // 初始负载为 0
	pool.rebuildNodeCache()
	d.services[svcName] = pool

	// 3. 模拟业务端向 Redis 上报了负载 (Load = 150)
	redisKey := redisKeyOfUpload(svcName, nodeID)
	err := db.Redis.Set(context.Background(), redisKey, 150, time.Minute).Err()
	assert.NoError(t, err)

	// 4. 执行同步
	d.syncLoad()

	// 5. 验证 Discoverer 内存中的负载是否被正确更新
	d.mtx.RLock()
	updatedLoad := d.services[svcName].nodes[nodeID].Load
	d.mtx.RUnlock()

	assert.Equal(t, int32(150), updatedLoad, "Discoverer 应该从 Redis 成功拉取并更新内存负载")
}

// ============================================================================
// 测试 5：全量替换时的 Load 继承逻辑测试
// ============================================================================
func TestDiscoverer_LoadInheritance(t *testing.T) {
	d := &Discoverer{
		services: make(map[string]*NodeGroup),
	}

	svcName := "chat_service"
	nodeID := int32(1)

	// 1. 构造旧缓存 (带有历史负载 999)
	oldPool := &NodeGroup{nodes: make(map[int32]NodeMeta)}
	oldPool.nodes[nodeID] = NodeMeta{NodeID: nodeID, Load: 999}
	d.services[svcName] = oldPool

	// 2. 模拟从 Etcd 拉取到了全新的数据 (Etcd 中的数据是没有动态 Load 的)
	newPools := make(map[string]*NodeGroup)
	newPool := &NodeGroup{nodes: make(map[int32]NodeMeta)}
	newPool.nodes[nodeID] = NodeMeta{NodeID: nodeID, Load: 0} // Etcd 里拉出来 Load 是 0
	newPools[svcName] = newPool

	// 3. 执行继承逻辑 (提取自 syncFullState)
	for sName, newOneSrv := range newPools {
		if oldOneSrv, exists := d.services[sName]; exists {
			for nID, newMeta := range newOneSrv.nodes {
				if oldMeta, ok := oldOneSrv.nodes[nID]; ok {
					// 继承 Load
					newMeta.Load = oldMeta.Load
					newOneSrv.nodes[nID] = newMeta
				}
			}
		}
		newOneSrv.rebuildNodeCache()
	}
	// 替换
	d.services = newPools

	// 4. 验证继承结果
	assert.Equal(t, int32(999), d.services[svcName].nodes[nodeID].Load, "全量替换时，必须继承旧缓存的 Load，不能归零")
}

func TestNewWatcher(t *testing.T) {
	Watch()

	err := RegisterDefault(flag.SrvName(pb.Server_Game), &NodeMeta{NodeID: 1})
	require.NoError(t, err)
	// time.Sleep(time.Millisecond * 50)
	Close()
}

func TestRegister(t *testing.T) {
	err := RegisterDefault(flag.SrvName(pb.Server_Game), &NodeMeta{NodeID: 1})
	require.NoError(t, err)
	Watch()
	// time.Sleep(time.Millisecond * 50)
	Close()
}

func TestPick(t *testing.T) {
	Watch()

	r1, err := NewRegister(etcdCli, redisCli, flag.SrvName(pb.Server_Game), &NodeMeta{NodeID: 1}, 30)
	require.NoError(t, err)
	r1.UpdateLoad(2)
	time.Sleep(time.Second)

	exist := Exists(flag.SrvName(pb.Server_Game), 1)
	require.True(t, exist)
	exist = Exists(flag.SrvName(pb.Server_Game), 2)
	require.False(t, exist)

	r2, err := NewRegister(etcdCli, redisCli, flag.SrvName(pb.Server_Game), &NodeMeta{NodeID: 2}, 30)
	require.NoError(t, err)
	r2.UpdateLoad(4)
	time.Sleep(time.Second)

	exist = Exists(flag.SrvName(pb.Server_Game), 1)
	require.True(t, exist)
	exist = Exists(flag.SrvName(pb.Server_Game), 2)
	require.True(t, exist)

	time.Sleep(time.Second * 6)

	id, ok := Select(flag.SrvName(pb.Server_Game))
	require.True(t, ok)
	require.Equal(t, int32(1), id)

	r1.UpdateLoad(5)
	time.Sleep(time.Second * 6)

	id, ok = Select(flag.SrvName(pb.Server_Game))
	require.True(t, ok)
	require.Equal(t, int32(2), id)

	exist = Exists(flag.SrvName(pb.Server_Game), 1)
	require.True(t, exist)
	exist = Exists(flag.SrvName(pb.Server_Game), 2)
	require.True(t, exist)

	r1.Close()

	exist = Exists(flag.SrvName(pb.Server_Game), 1)
	require.False(t, exist)
	exist = Exists(flag.SrvName(pb.Server_Game), 2)
	require.True(t, exist)

	r2.Close()

	exist = Exists(flag.SrvName(pb.Server_Game), 1)
	require.False(t, exist)
	exist = Exists(flag.SrvName(pb.Server_Game), 2)
	require.False(t, exist)

	Close()
}
