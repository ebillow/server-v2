package save_proto

import (
	"context"
	"os"
	"server/api/pb"
	"server/pkg/db"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMain(m *testing.M) {
	if err := db.InitRedis(db.RedisCfg{
		Addr:     []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
		Password: "",
		DB:       0,
		Name:     "",
	}, 0); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestSaveProto(t *testing.T) {
	data := pb.S2SReqLogin{
		SesID:       10,
		RoleID:      230,
		ReConnToken: 3123123,
		Seq:         233,
	}
	b, err := proto.Marshal(&data)
	require.NoError(t, err)

	err = db.Redis.Set(context.Background(), "test_proto_save", string(b), time.Minute*50).Err()
	require.NoError(t, err)

	data2 := pb.S2SResLogin{
		Res: &pb.S2CLogin{
			Code:            233,
			GameID:          1,
			OpenTime:        12312312,
			ServerNowTime:   123123,
			ServerBeginTime: 1233,
		},
		GameID: 230,
	}
	b2, err := proto.Marshal(&data2)
	require.NoError(t, err)

	err = db.Redis.HSet(context.Background(), "test_proto_save_h", "s2s_req", string(b), "s2s_res", string(b2)).Err()
	db.Redis.Expire(context.Background(), "test_proto_save_h", time.Minute*10)
	require.NoError(t, err)
}
