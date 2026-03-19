package login_mgr

import (
	"context"
	"encoding/json"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/proto"
	"server/pkg/db"
	"server/pkg/pb"
	"server/pkg/util"
	"testing"
	"time"
)

func mock() *pb.S2CLogin {
	return &pb.S2CLogin{
		Code:            0,
		GameID:          23,
		OpenTime:        time.Now().Unix(),
		ServerNowTime:   time.Now().Unix(),
		ServerBeginTime: time.Now().Unix(),
		Dev:             "adfadsfadsf",
		RetDesc:         "adsfadsfadsf",
		Token:           util.RandToken(),
		CliVersion:      "1.1.0",
		Player: &pb.RoleData{
			ID:             1111,
			Level:          33,
			Exp:            23123,
			Name:           "adsfadsf",
			Country:        3,
			OfflineTime:    time.Now().Unix(),
			OnlineTime:     time.Now().Unix(),
			CreateTime:     time.Now().Unix(),
			ResetTime:      time.Now().Unix(),
			DayChange:      1,
			DataResetMonth: 2,
		},
	}
}

// BenchmarkPB-10    	 1748578	       679.6 ns/op
func BenchmarkPB(b *testing.B) {
	d := mock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bts, err := proto.Marshal(d)
		require.NoError(b, err)
		err = proto.Unmarshal(bts, d)
		require.NoError(b, err)
	}
}

// BenchmarkBson-10    	  391648	      3103 ns/op
func BenchmarkBson(b *testing.B) {
	d := mock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bts, err := bson.Marshal(d)
		require.NoError(b, err)
		err = bson.Unmarshal(bts, d)
		require.NoError(b, err)
	}
}

// BenchmarkJson-10    	  346207	      3403 ns/op
func BenchmarkJson(b *testing.B) {
	d := mock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bts, err := json.Marshal(d)
		require.NoError(b, err)
		err = json.Unmarshal(bts, d)
		require.NoError(b, err)
	}
}

// BenchmarkJsoniter-10    	  565819	      2169 ns/op
func BenchmarkJsoniter(b *testing.B) {
	d := mock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bts, err := jsoniter.Marshal(d)
		require.NoError(b, err)
		err = jsoniter.Unmarshal(bts, d)
		require.NoError(b, err)
	}
}

// // BenchmarkMsgPack-10    	  659154	      1767 ns/op
// func BenchmarkMsgPack(b *testing.B) {
// 	d := mock()
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		bts, err := msgpack.Marshal(d)
// 		require.NoError(b, err)
// 		err = msgpack.Unmarshal(bts, d)
// 		require.NoError(b, err)
// 	}
// }

func TestMsgLen(t *testing.T) {
	d := mock()
	bts, err := json.Marshal(d)
	require.NoError(t, err)
	t.Log("json", len(bts))
	// bts, err = msgpack.Marshal(d)
	// require.NoError(t, err)
	// t.Log("msgpack", len(bts))
	bts, err = proto.Marshal(d)
	require.NoError(t, err)
	t.Log("proto", len(bts))
}

func TestBsonSave(t *testing.T) {
	d := pb.RoleData{
		ID:    2,
		Level: 100,
		Exp:   9999,
		Name:  "testName",
		Items: map[string]int64{"Gold": 888, "ItemA": 5555},
	}
	b, err := bson.Marshal(&d)
	require.NoError(t, err)
	t.Log(string(b))

	db.Redis.Set(context.Background(), "test:bson", string(b), time.Minute)

	b2 := db.Redis.Get(context.Background(), "test:bson").Val()
	d2 := pb.RoleData{}
	err = bson.Unmarshal([]byte(b2), &d2)
	require.NoError(t, err)
	t.Log(&d2)
}

// func TestMsgPackSave(t *testing.T) {
// 	d := pb.RoleData{
// 		ID:    2,
// 		Level: 100,
// 		Exp:   9999,
// 		Name:  "testName",
// 		Items: map[string]int64{"Gold": 888, "ItemA": 5555},
// 	}
// 	b, err := msgpack.Marshal(&d)
// 	require.NoError(t, err)
// 	t.Log(string(b))
//
// 	db.Redis.Set(context.Background(), "test:bson", string(b), time.Minute)
//
// 	b2 := db.Redis.Get(context.Background(), "test:bson").Val()
// 	d2 := pb.RoleData{}
// 	err = msgpack.Unmarshal([]byte(b2), &d2)
// 	require.NoError(t, err)
// 	t.Log(&d2)
// }
