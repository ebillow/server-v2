package onlines

import (
	"sync"
	"testing"
)

func TestInit(t *testing.T) {
	for i, shard := range roleShards {
		if shard == nil {
			t.Fatalf("shard %d is nil, initialization failed", i)
		}
		if shard.roles == nil {
			t.Fatalf("shard %d roles map is nil", i)
		}
	}
}

// TestAddAndGet 测试基础的添加和查询功能
func TestAddAndGet(t *testing.T) {

	roleID := uint64(1001)
	expectedGameID := uint8(5)

	Add(roleID, Data{SesID: 2001, GameID: expectedGameID})

	// 验证获取存在的 Role
	gameID, ok := GetGameID(roleID)
	if !ok {
		t.Errorf("expected to find role %d", roleID)
	}
	if gameID != expectedGameID {
		t.Errorf("expected gameID %d, got %d", expectedGameID, gameID)
	}

	// 验证获取不存在的 Role
	_, ok = GetGameID(9999)
	if ok {
		t.Errorf("should not find non-existent role 9999")
	}
}

// TestDel 测试删除功能
func TestDel(t *testing.T) {

	roleID := uint64(1002)
	Add(roleID, Data{SesID: 2002, GameID: 1})

	// 确认添加成功
	if _, ok := GetGameID(roleID); !ok {
		t.Fatalf("setup failed, role %d not added", roleID)
	}

	// 删除
	Remove(roleID)

	// 确认删除成功
	if _, ok := GetGameID(roleID); ok {
		t.Errorf("role %d should have been deleted", roleID)
	}

	// 重复删除不应该 panic
	Remove(roleID)
}

// TestConcurrentStress 高并发压力测试 (读写混合)
// 运行命令: go test -v -race -run TestConcurrentStress
func TestConcurrentStress(t *testing.T) {
	var wg sync.WaitGroup
	const numWorkers = 100
	const opsPerWorker = 1000

	// 启动 100 个 Goroutine 并发执行 Add, Get, Del
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < opsPerWorker; j++ {
				// 构造 RoleID，利用 workerID 保证一定范围内的离散性
				roleID := uint64(workerID*opsPerWorker + j)
				gameID := uint8(j % 255)

				// 1. 并发写入
				Add(roleID, Data{SesID: roleID + 10000, GameID: gameID})

				// 2. 并发读取
				if gotGameID, ok := GetGameID(roleID); !ok {
					t.Errorf("concurrent GetGameID failed for role %d", roleID)
				} else if gotGameID != gameID {
					t.Errorf("concurrent GetGameID value mismatch: got %d, want %d", gotGameID, gameID)
				}

				// 3. 并发删除 (删除一半的数据，保留一半)
				if j%2 == 0 {
					Remove(roleID)
					if _, ok := GetGameID(roleID); ok {
						t.Errorf("concurrent Del failed for role %d", roleID)
					}
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestBitwiseModulo 验证位运算取模的正确性
func TestBitwiseModulo(t *testing.T) {
	// 确保 shardCount 是 2 的幂次方，否则位运算 & (shardCount-1) 会导致严重的数据倾斜或越界
	if shardCount&(shardCount-1) != 0 {
		t.Fatalf("shardCount (%d) MUST be a power of 2 for bitwise modulo to work", shardCount)
	}

	// 验证越界情况
	testIDs := []uint64{0, 1023, 1024, 9999999}
	for _, id := range testIDs {
		index := id & (shardCount - 1)
		if index >= shardCount {
			t.Errorf("index %d out of bounds for roleID %d", index, id)
		}
	}
}
