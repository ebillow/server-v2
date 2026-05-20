package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// setupEtcd 初始化 Etcd 客户端。如果本地没有运行 Etcd，则跳过测试。
func setupEtcd(t *testing.T) *clientv3.Client {
	// 初始化日志，方便看测试过程中的输出
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 2 * time.Second,
	})
	require.NoError(t, err)

	// 探测 Etcd 是否可用，不可用则直接跳过测试
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = cli.Status(ctx, "127.0.0.1:2379")
	if err != nil {
		t.Skip("本地 Etcd 未启动，跳过选主测试: ", err)
	}

	return cli
}

// waitMasterStatus 等待 isMaster 变为期望的值 (带超时机制，防止测试死锁)
func waitMasterStatus(t *testing.T, expected bool, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if IsMaster() == expected {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("超时! 期望 IsMaster() == %v, 但实际为 %v", expected, IsMaster())
}

// 测试场景 1：正常当选与主动卸任
func TestElection_SingleNode(t *testing.T) {
	cli := setupEtcd(t)
	defer cli.Close()

	// 确保初始状态为 false
	isMaster.Store(false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 启动选举
	go StartElection(ctx, cli, "test-service", "node-1", 5)

	// 2. 验证是否成功当选为 Master (最多等 3 秒)
	waitMasterStatus(t, true, 3*time.Second)
	require.True(t, IsMaster(), "节点应该成为 Master")

	// 3. 模拟程序退出，取消 context
	cancel()

	// 4. 验证是否成功卸任 Master
	waitMasterStatus(t, false, 3*time.Second)
	require.False(t, IsMaster(), "Context 取消后，节点应该卸任 Master")
}

// 测试场景 2：多节点故障转移 (Failover) 模拟
func TestElection_Failover(t *testing.T) {
	cli := setupEtcd(t)
	defer cli.Close()

	isMaster.Store(false)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2() // ctx1 会在中间被主动 cancel

	// 1. 启动节点 1
	go StartElection(ctx1, cli, "failover-service", "node-1", 5)

	// 等待节点 1 成为 Master
	waitMasterStatus(t, true, 3*time.Second)

	// 2. 启动节点 2 (此时节点 1 是 Master，节点 2 会阻塞等待)
	// 注意：因为我们用的是全局变量 isMaster，所以这里主要是测试节点 2 的 Campaign 是否正常阻塞，
	// 以及节点 1 退出后，节点 2 是否能继续维持 isMaster = true。
	go StartElection(ctx2, cli, "failover-service", "node-2", 5)

	// 稍微等一下，确保节点 2 已经进入阻塞状态
	time.Sleep(1 * time.Second)
	require.True(t, IsMaster(), "节点 1 依然应该是 Master")

	// 3. 模拟节点 1 崩溃/退出
	cancel1()

	// 此时节点 1 会卸任 (isMaster 短暂变为 false)，随后节点 2 被唤醒接管 (isMaster 再次变为 true)
	// 我们直接等待最终结果：isMaster 必须再次稳定为 true
	waitMasterStatus(t, true, 4*time.Second)
	require.True(t, IsMaster(), "节点 1 退出后，节点 2 应该接管成为 Master")
}
