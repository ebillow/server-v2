package msgq

import (
	"os"
	pb "server/api/pb"
	"server/pkg/cfg"
	"server/pkg/gnet/gmsg"
	"server/pkg/logger"
	"server/pkg/util"
	"testing"
	"time"

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

	err := Q.Init("nats://localhost:4222,nats://localhost:4222", pb.Server_Game, 1, nats.UserInfo("123456", "123456"))
	if err != nil {
		panic(err)
	}
	SrvRpc()
	os.Exit(m.Run())
}

type actor struct {
	c chan gmsg.Message
}

func (a *actor) Run() {
	data := pb.S2SReqLogin{}
	for {
		select {
		case ctx := <-a.c:
			err := proto.Unmarshal(ctx.Data, &data)
			// zap.L().Info("msg recv", zap.Any("msg", &data))
			err = RpcRespond(&Q, ctx.Reply(), &pb.S2SResLogin{
				GameID: 111,
				Res: &pb.S2CLogin{
					Token: data.Seq,
				},
			})
			data.Reset()
			if err != nil {
				panic(err)
			}
		}
	}
}

func SrvRpc() {
	chs := make([]*actor, 0)
	for i := 0; i < 3; i++ {
		a := &actor{make(chan gmsg.Message, 128)}
		chs = append(chs, a)
		go a.Run()
	}
	err := Q.Serve(func(ctx gmsg.Message) {
		chs[util.RandRange(0, 3)].c <- ctx
	})
	if err != nil {
		panic(err)
	}
}

func TestRpcCall(t *testing.T) {
	for i := 0; i < 10000; i++ {
		token := uint32(i * 10)
		ack := pb.S2SResLogin{}
		err := RpcCall(&Q, &pb.S2SReqLogin{
			SesID:       1,
			RoleID:      1,
			ReConnToken: 1231231,
			Seq:         token,
		}, &ack, pb.Server_Game, 1, 1, time.Millisecond*500)
		require.NoError(t, err)
		require.Equal(t, ack.Res.Token, token)
		// t.Log(ack)
	}
}
