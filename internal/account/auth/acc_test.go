package auth

import (
	"context"
	"server/api/pb"
	"server/internal/share/model"
	"server/pkg/db"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAccountSave(t *testing.T) {
	ctx := context.Background()

	acc := &Account{
		AccID: 33,
		Binds: []string{FormatBindKey(pb.SdkType_Guest, "test")},
	}

	db.Redis.HSet(ctx, model.KeyAccount(acc.AccID), acc.FieldBinds(), acc.MarshalBinds())

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

func TestAccountSaveDB(t *testing.T) {
	ctx := context.Background()

	accs := make([]*Account, 0)
	accs = append(accs, &Account{
		AccID: 111,
		Binds: []string{FormatBindKey(pb.SdkType_Guest, "test_device1"), FormatBindKey(pb.SdkType_Google, "test_google1")},
	})
	accs = append(accs, &Account{
		AccID: 222,
		Binds: []string{FormatBindKey(pb.SdkType_Guest, "test_device2"), FormatBindKey(pb.SdkType_Google, "test_google2")},
	})
	accs = append(accs, &Account{
		AccID: 333,
		Binds: []string{FormatBindKey(pb.SdkType_Guest, "test_device3"), FormatBindKey(pb.SdkType_Google, "test_google3")},
	})

	for _, acc := range accs {
		_, err := db.MongoDB().Collection(acc.CollectionName()).InsertOne(ctx, acc)
		require.NoError(t, err)
	}
}

func TestAccountLoadFromDB(t *testing.T) {
	ctx := context.Background()
	acc := &Account{}

	binds := []string{FormatBindKey(pb.SdkType_Guest, "test_device1")}
	filter := bson.M{"binds": bson.M{"$in": binds}}
	err := db.MongoDB().Collection(acc.CollectionName()).FindOne(ctx, filter).Decode(acc)
	require.NoError(t, err)
	require.Equal(t, acc.AccID, uint64(111))

	binds = []string{FormatBindKey(pb.SdkType_Google, "test_google2")}
	filter = bson.M{"binds": bson.M{"$in": binds}}
	err = db.MongoDB().Collection(acc.CollectionName()).FindOne(ctx, filter).Decode(acc)
	require.NoError(t, err)
	require.Equal(t, acc.AccID, uint64(222))

	binds = []string{FormatBindKey(pb.SdkType_Guest, "test_device3")}
	filter = bson.M{"binds": bson.M{"$in": binds}}
	err = db.MongoDB().Collection(acc.CollectionName()).FindOne(ctx, filter).Decode(acc)
	require.NoError(t, err)
	require.Equal(t, acc.AccID, uint64(333))

	binds = []string{FormatBindKey(pb.SdkType_Guest, "test_device4")}
	filter = bson.M{"binds": bson.M{"$in": binds}}
	err = db.MongoDB().Collection(acc.CollectionName()).FindOne(ctx, filter).Decode(acc)
	require.Error(t, err)

	binds = []string{FormatBindKey(pb.SdkType_Facebook, "test_device1")}
	filter = bson.M{"binds": bson.M{"$in": binds}}
	err = db.MongoDB().Collection(acc.CollectionName()).FindOne(ctx, filter).Decode(acc)
	require.Error(t, err)

	binds = []string{FormatBindKey(pb.SdkType_Google, "test_google2"),
		FormatBindKey(pb.SdkType_Guest, "test_device1"),
		FormatBindKey(pb.SdkType_Guest, "test_device3"),
	}
	filter = bson.M{"binds": bson.M{"$in": binds}}
	ret, err := db.MongoDB().Collection(acc.CollectionName()).Find(ctx, filter)
	require.NoError(t, err)
	for ret.Next(ctx) {
		err = ret.Decode(&acc)
		require.NoError(t, err)
		t.Log(acc.AccID)
	}
}
