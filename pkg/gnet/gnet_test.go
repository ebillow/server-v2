package gnet

import (
	pb "server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/cfg"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/msgq"
	"server/pkg/logger"
	"server/pkg/util"
	"sync"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMain(m *testing.M) {
	cfg.Load("127.0.0.1:2379", "local")

	logger.NewZapLog("../../../bin/logger/test.logger", logger.Config{
		Level:   0,
		Console: true,
	})

	m.Run()
}

func TestSendAndServe(t *testing.T) {
	err := msgq.Q.Init("nats://localhost:4222,nats://localhost:4222", pb.Server_Game, 1, nats.UserInfo("123456", "123456"))
	require.NoError(t, err)

	wait := sync.WaitGroup{}
	data := pb.S2SReqLogin{}
	cnt := 0
	err = msgq.Q.Serve(func(ctx gctx.Context) {
		err = proto.Unmarshal(ctx.Data, &data)
		require.NoError(t, err)
		require.Equal(t, data.SesID, data.RoleID*2)
		require.Equal(t, data.ReConnToken, data.RoleID*3)
		require.Equal(t, data.Seq, uint32(data.RoleID*4))

		// zap.L().Info("msg recv", zap.Any("msg", &data))
		data.Reset()
		wait.Done()
		cnt++
	})
	require.NoError(t, err)

	for i := 0; i < 1000000; i++ {
		wait.Add(2)
		SendToGame(1, msgid.MsgIDS2S_S2SReqLogin, &pb.S2SReqLogin{
			SesID:       uint64(i * 2),
			RoleID:      uint64(i),
			ReConnToken: uint64(i * 3),
			Seq:         uint32(i * 4),
		}, 0, 0)
		SendToGameS(1, &pb.S2SReqLogin{
			SesID:       uint64(i * 2),
			RoleID:      uint64(i),
			ReConnToken: uint64(i * 3),
			Seq:         uint32(i * 4),
		}, 0, 0)
	}
	wait.Wait()
	t.Log("run ", cnt)
}

func TestVTProto(t *testing.T) {
	for i := 0; i < 10000; i++ {
		data := pb.S2SReqLogin{
			Req: &pb.C2SLogin{
				SdkType: 111,
				Account: "31231",
				Token:   "23dsfadf",
				Channel: util.RandRange(uint32(1000), 100000),
				Dev:     "adfadsf",
				Area:    123,
				Version: "1.0.0",
			},
			SesID:       2220,
			RoleID:      1231230,
			ReConnToken: util.RandRange(uint64(1000), 100000),
			Seq:         11111,
			BindAcc:     "23123",
		}
		b, err := proto.Marshal(&data)
		require.NoError(t, err)

		d2 := pb.S2SReqLogin{}
		err = d2.UnmarshalVT(b)
		require.NoError(t, err)
		require.Equal(t, data.ReConnToken, d2.ReConnToken)
		require.Equal(t, data.Req.Channel, d2.Req.Channel)

		b2, err := data.MarshalVT()
		require.NoError(t, err)

		d3 := pb.S2SReqLogin{}
		err = proto.Unmarshal(b2, &d3)
		require.NoError(t, err)

		require.Equal(t, data.ReConnToken, d3.ReConnToken)
		require.Equal(t, data.Req.Channel, d3.Req.Channel)
	}
}
