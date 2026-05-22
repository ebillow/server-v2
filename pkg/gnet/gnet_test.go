package gnet

import (
	pb2 "server/api/pb"
	"server/pkg/cfg"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/msgq"
	"server/pkg/logger"
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
	err := msgq.Q.Init("nats://localhost:4222,nats://localhost:4222", pb2.Server_Game, 1, nats.UserInfo("123456", "123456"))
	require.NoError(t, err)

	wait := sync.WaitGroup{}
	data := pb2.S2SReqLogin{}
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
		wait.Add(1)
		SendToGame(1, &pb2.S2SReqLogin{
			SesID:       uint64(i * 2),
			RoleID:      uint64(i),
			ReConnToken: uint64(i * 3),
			Seq:         uint32(i * 4),
		}, 0, 0)
	}
	wait.Wait()
	t.Log("run ", cnt)
}
