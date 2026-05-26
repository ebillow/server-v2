package auth

import (
	"context"
	"server/internal/share/model"
	"server/pkg/db"
	"testing"
)

func TestAccountSave(t *testing.T) {
	ctx := context.Background()

	acc := &Account{
		AccID:  33,
		Device: "test",
	}
	db.Redis.HSet(ctx, model.KeyAccount(acc.AccID), acc.FieldDevice(), acc.Device)

	success, err := acc.SaveLoginData(ctx, 0)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if !success {
		t.Fatal("first save should succeed")
	}
}
func TestAccount_CAS_Lock(t *testing.T) {
	ctx := context.Background()

	acc := &Account{
		AccID:  1000000,
		GameID: 1,
		Seq:    1,
		Passwd: 123456,
	}

	// 1. 首次保存，Redis 中 seq 为空（视为 "0"），期望 oldSeq=0
	success, err := acc.SaveLoginData(ctx, 0)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if !success {
		t.Fatal("first save should succeed")
	}

	// 2. 模拟正常流程：读取旧 Seq=1，更新 Seq=2
	acc.Seq = 2
	acc.GameID = 2
	success, err = acc.SaveLoginData(ctx, 1)
	if err != nil || !success {
		t.Fatal("second save with correct oldSeq should succeed")
	}

	// 3. 模拟并发冲突（脑裂）：拿着过期的 oldSeq=1 尝试覆盖
	acc.GameID = 99 // 尝试改成错误的游戏服
	success, err = acc.SaveLoginData(ctx, 1)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}
	if success {
		t.Fatal("CAS failed! Allowed overwrite with stale Seq")
	}

	// 4. 测试 Logout 清除 GameID
	success, err = acc.ClearGameID(ctx, 2) // 拿着正确的 Seq=2 去清空
	if err != nil || !success {
		t.Fatal("ClearGameID should succeed")
	}
}
