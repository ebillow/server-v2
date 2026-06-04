package session

import (
	"context"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/gnet"
	"server/pkg/idgen"
	"server/pkg/queue"
	"server/pkg/thread"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	maxRecvMsgSize = 10240
)

type MsgSend struct {
	ID   uint32
	Data []byte
}

// Session 客户端和gate的网络会话
type Session struct {
	Id            uint64
	GameID        atomic.Int32
	conn          *websocket.Conn
	Ip            string
	disConnReason atomic.Int32

	out *queue.SwapQueue[MsgSend]

	cancel    context.CancelFunc
	closeOnce sync.Once
}

func (s *Session) UpdateSerId(gameID int32) {
	s.GameID.Store(gameID)
}

// GracefulStop 关闭,线程安全
func (s *Session) Close(why pb.DisconnectReason) {
	s.closeOnce.Do(func() {
		s.disConnReason.Store(int32(why))
		s.cancel()

		Remove(s.Id)
		gnet.SendToGame(s.getSerID(pb.Server_Game), msgid.MsgIDS2S_S2SGt2SDisconnect, &pb.S2SGt2SDisconnect{
			SesID: s.Id,
			Why:   pb.DisconnectReason(s.disConnReason.Load()),
		}, 0, 0)
		if s.conn != nil {
			_ = s.conn.Close()
		}
		zap.L().Info("disconnect", zap.Inline(s))
	})
}

func (s *Session) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("s.ip", s.Ip)
	encoder.AddUint64("s.id", s.Id)
	return nil
}

// start
func (s *Session) start() {
	s.out = queue.NewSwapQueue[MsgSend](netCfg.OutChanSize, netCfg.OutChanSize*100)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	id, _ := idgen.Gen()
	s.Id = uint64(id)
	Add(s.Id, s)

	waitGroup.Add(2)
	thread.GoSafe(func() {
		s.sendLoop(ctx)
	})
	thread.GoSafe(func() {
		s.readLoop(ctx, netCfg)
	})
	zap.L().Info("connect", zap.Inline(s))
}

func (s *Session) getSerID(ser pb.Server) uint8 {
	return uint8(s.GameID.Load())
}
