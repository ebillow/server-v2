package js

import (
	"context"
	"fmt"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/gnet/gctx"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
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
	S.Send(pb.Server_Gateway, 1, uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("hello world"), 0, 0)
}

func TestSub(t *testing.T) {
	wg := sync.WaitGroup{}
	maxCnt := 30000
	wg.Add(maxCnt)
	// ret := make(map[int64]int)
	// var mtx sync.Mutex

	for i := 0; i < 3; i++ {
		err := S.Serve(context.Background(), func(msg gctx.Context) {
			// fmt.Println(i, msg, string(msg.Data))
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
		S.Send(pb.Server_Gateway, 1, uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("to index: hello world"), 0, 0)
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
		err := S.Init(pb.Server_Gateway, uint8(i), strings.Join(servers, ","), nats.UserInfo("123456", "123456"))
		require.NoError(t, err)

		err = S.Serve(context.Background(), func(msg gctx.Context) {
			fmt.Println(i, msg, string(msg.Data))
			wg.Done()
		})

		require.NoError(t, err)
	}

	for i := 0; i < maxCnt; i++ {
		S.Send(pb.Server_Gateway, uint8(i), uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("to index: hello world"), 3, 220)

		S.SendAny(pb.Server_Gateway, uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("to any: hello world"), 440, 20)
	}

	wg.Wait()
	S.Shutdown()
}

func TestPull(t *testing.T) {
	p, err := NewPullConsumer(context.Background(), &S, getIndexSubject(pb.Server_Gateway, 0))
	require.NoError(t, err)
	wg := sync.WaitGroup{}
	p.Start(context.Background(), func(ctx gctx.Context) {

		t.Log(ctx, string(ctx.Data))
		wg.Done()
	})

	cnt := 10
	wg.Add(cnt)
	for i := 0; i < cnt; i++ {
		S.Send(pb.Server_Gateway, 0, uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("to idx: hello world"), 440, 20)
	}
	wg.Wait()
}

func TestPullMulti(t *testing.T) {
	wg := sync.WaitGroup{}

	recv := make(map[int]*atomic.Int32)

	for i := 0; i < 3; i++ {
		recv[i] = &atomic.Int32{}
		p, err := NewPullConsumer(context.Background(), &S, getIndexSubject(pb.Server_Gateway, 0))
		require.NoError(t, err)
		p.Start(context.Background(), func(ctx gctx.Context) {
			// t.Log(i, v, string(v.Data))
			recv[i].Add(1)
			wg.Done()

		})
	}

	cnt := 1000000
	wg.Add(cnt)
	for i := 0; i < cnt; i++ {
		S.Send(pb.Server_Gateway, 0, uint32(msgid.MsgIDS2S_S2SKickAcc), []byte("to idx: hello world"), 440, 20)
	}
	wg.Wait()
	for k, v := range recv {
		t.Log(k, v.Load())
	}
}
