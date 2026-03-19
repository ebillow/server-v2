package js

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"strings"
	"sync"
	"testing"
)

func TestMain(m *testing.M) {
	servers := []string{"nats://127.0.0.1:4223"}
	err := S.Init(pb.Server_Gateway, 1, strings.Join(servers, ","), nats.UserInfo("123456", "123456"))
	if err != nil {
		panic(err)
	}
	m.Run()
}

func TestPub(t *testing.T) {
	err := S.Send(pb.Server_Gateway, 1, uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("hello world"), 0, 0)
	require.NoError(t, err)
}

func TestSub(t *testing.T) {
	wg := sync.WaitGroup{}
	maxCnt := 3
	wg.Add(maxCnt)
	// ret := make(map[int64]int)
	// var mtx sync.Mutex

	for i := 0; i < 3; i++ {
		err := S.Serve(context.Background(), func(msg *pb.NatsMsg) {
			fmt.Println(i, msg)
			// mtx.Lock()
			// if _, ok := ret[msg.RoleID]; ok {
			// 	t.Fatalf("repeated msg %d", msg.RoleID)
			// }
			// ret[msg.RoleID] = i
			// mtx.Unlock()
			wg.Done()
		})
		require.NoError(t, err)
	}

	for i := 0; i < maxCnt; i++ {
		err := S.Send(pb.Server_Gateway, 1, uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("to index: hello world"), 0, 0)
		require.NoError(t, err)
	}

	wg.Wait()

	S.Shutdown()
}

func TestMultiSub(t *testing.T) {
	wg := sync.WaitGroup{}
	maxCnt := 3
	wg.Add(maxCnt * 2)
	for i := 0; i < 3; i++ {
		servers := []string{"nats://127.0.0.1:4223"}
		err := S.Init(pb.Server_Gateway, int32(i), strings.Join(servers, ","), nats.UserInfo("123456", "123456"))
		require.NoError(t, err)

		err = S.Serve(context.Background(), func(msg *pb.NatsMsg) {
			fmt.Println(i, msg)
			wg.Done()
		})

		require.NoError(t, err)
	}

	for i := 0; i < maxCnt; i++ {
		err := S.Send(pb.Server_Gateway, int32(i), uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("to index: hello world"), 3, 220)
		require.NoError(t, err)

		err = S.SendAny(pb.Server_Gateway, uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("to any: hello world"), 440, 20)
		require.NoError(t, err)
	}

	wg.Wait()
	S.Shutdown()
}
