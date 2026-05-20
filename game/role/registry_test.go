package role

import (
	"context"
	"fmt"
	"server/pkg/queue"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockRole 辅助函数：创建一个用于测试的模拟 Role
func mockRole() *Role {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Role{
		Events: queue.NewSwapQueue[Event](EventChanSize, EventChanSize*100),
		Ctx:    ctx,
		Cancel: cancel,
	}

	// 模拟 Role 的主循环，用于测试 KickAndWait 和 CloseAndWait
	r.Wait.Add(1)
	go func() {
		defer r.Wait.Done()
		<-r.Ctx.Done() // 阻塞直到被 Cancel (Kick)
	}()

	return r
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if len(reg.roleShards) != shardCount {
		t.Errorf("expected %d role shards, got %d", shardCount, len(reg.roleShards))
	}
	if len(reg.sesShards) != shardCount {
		t.Errorf("expected %d ses shards, got %d", shardCount, len(reg.sesShards))
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	role1 := mockRole()
	defer role1.Cancel()

	reg.Register(1001, 2001, role1)

	// 1. 测试按 RoleID 获取
	m, ok := reg.get(1001)
	if !ok {
		t.Fatalf("expected to find role 1001")
	}
	if m.ctx != role1.Ctx {
		t.Errorf("context mismatch")
	}

	// 2. 测试按 SesID 获取
	m2, ok := reg.getBySes(2001)
	if !ok {
		t.Fatalf("expected to find role by ses 2001")
	}
	if m2.ctx != role1.Ctx {
		t.Errorf("context mismatch")
	}

	// 3. 测试获取不存在的 ID
	if _, ok := reg.get(9999); ok {
		t.Errorf("should not find non-existent role")
	}
	if _, ok := reg.getBySes(9999); ok {
		t.Errorf("should not find non-existent session")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()
	role1 := mockRole()
	defer role1.Cancel()

	reg.Register(1001, 2001, role1)

	// 1. 测试错位解绑（恶意或延迟的解绑请求）
	// 尝试用错误的 SesID 解绑
	reg.Unregister(1001, 9999)
	if _, ok := reg.get(1001); !ok {
		t.Errorf("role 1001 should not be deleted due to mismatched sesID")
	}

	// 尝试用错误的 RoleID 解绑正确的 SesID
	reg.Unregister(9999, 2001)
	if _, ok := reg.getBySes(2001); !ok {
		t.Errorf("session 2001 should not be deleted due to mismatched roleID")
	}

	// 2. 测试正常解绑
	reg.Unregister(1001, 2001)
	if _, ok := reg.get(1001); ok {
		t.Errorf("role 1001 should be deleted")
	}
	if _, ok := reg.getBySes(2001); ok {
		t.Errorf("session 2001 should be deleted")
	}
}

func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry()
	for i := 1; i <= 50; i++ {
		r := mockRole()
		reg.Register(uint64(i), uint64(i+1000), r)
	}

	if count := reg.Count(); count != 50 {
		t.Errorf("expected count 50, got %d", count)
	}
}

func TestRegistry_CloseAndWait(t *testing.T) {
	reg := NewRegistry()
	for i := 1; i <= 5; i++ {
		reg.Register(uint64(i), uint64(i+1000), mockRole())
	}

	// 记录开始时间，确保 CloseAndWait 能正常阻塞并最终释放
	start := time.Now()

	// CloseAndWait 会触发所有 mockRole 的 Cancel，从而让它们的 goroutine 退出并执行 Wait.Done()
	reg.CloseAndWait()

	duration := time.Since(start)
	if duration > time.Second {
		t.Errorf("CloseAndWait took too long: %v, possible deadlock", duration)
	}

	// 验证 context 是否都已结束
	for i := 1; i <= 5; i++ {
		m, ok := reg.get(uint64(i))
		if ok {
			select {
			case <-m.ctx.Done():
				// 正常，已关闭
			default:
				t.Errorf("role %d context was not canceled", i)
			}
		}
	}
}

// 核心：高并发压力测试（检查死锁和数据竞争）
// 运行命令: go test -v -race -run TestRegistry_Concurrent_Stress
func TestRegistry_Concurrent_Stress(t *testing.T) {
	reg := NewRegistry()
	var wg sync.WaitGroup

	const numWorkers = 100
	const opsPerWorker = 1000

	var opsCount int64

	// 启动 100 个并发 Goroutine
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < opsPerWorker; j++ {
				roleID := uint64(workerID*opsPerWorker + j)
				sesID := roleID + 100000

				// 1. 并发写入
				r := mockRole()
				reg.Register(roleID, sesID, r)

				// 2. 并发读取
				if _, ok := reg.get(roleID); !ok {
					t.Errorf("concurrent get failed for role %d", roleID)
				}
				if _, ok := reg.getBySes(sesID); !ok {
					t.Errorf("concurrent getBySes failed for ses %d", sesID)
				}

				// 3. 并发 Count
				if j%100 == 0 {
					_ = reg.Count()
				}

				// 4. 并发删除
				reg.Unregister(roleID, sesID)

				// 顺便清理后台 goroutine
				r.Cancel()

				atomic.AddInt64(&opsCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if reg.Count() != 0 {
		t.Errorf("expected registry to be empty after stress test, got %d", reg.Count())
	}
	fmt.Printf("Stress test completed: %d operations\n", opsCount)
}
