package msgq

import (
	"os"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMain(m *testing.M) {
	err := Q.Init("nats://localhost:4222,nats://localhost:4222", pb.Server_Game, 1, nats.UserInfo("123456", "123456"))
	if err != nil {
		panic(err)
	}

	data := pb.S2SReqLogin{}
	err = Q.Serve(func(natMsg *pb.NatsMsg, msg *nats.Msg) {
		err = proto.Unmarshal(natMsg.Data, &data)
		// zap.L().Info("msg recv", zap.Any("msg", &data))
		err = RpcRespond(msg, &pb.S2SResLogin{
			GameID: 111,
			Res: &pb.S2CLogin{
				Token: data.Seq,
			},
		})
		data.Reset()
		if err != nil {
			panic(err)
		}
	})
	os.Exit(m.Run())
}

func TestRpcCall(t *testing.T) {
	for i := 0; i < 1000; i++ {
		token := uint32(i * 10)
		ack, err := RpcCall[pb.S2SResLogin](Q, uint32(msgid.MsgIDS2S_S2SReqLogin), &pb.S2SReqLogin{
			SesID:       1,
			RoleID:      1,
			ReConnToken: 1231231,
			Seq:         token,
		}, pb.Server_Game, 1, 1, 0, time.Second*3)
		require.NoError(t, err)
		require.Equal(t, ack.Res.Token, token)
		// t.Log(ack)
	}
}
