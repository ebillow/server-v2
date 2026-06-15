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
	Id     uint64
	GameID atomic.Int32
	conn   *websocket.Conn
	Ip     string

	out *queue.SwapQueue[MsgSend]

	disConnReason atomic.Int32
	cancel        context.CancelFunc
	closeOnce     sync.Once
	loopCount     atomic.Int32
}

func (s *Session) Close(why pb.DisconnectReason) {
	s.closeOnce.Do(func() {
		s.disConnReason.Store(int32(why))
		s.cancel()
	})
}

func (s *Session) onLoopExit() {
	waitGroup.Done()

	if s.loopCount.Add(-1) == 0 {
		Remove(s.Id)

		reason := pb.DisconnectReason(s.disConnReason.Load())
		gnet.SendToGame(
			s.getSerID(pb.Server_Game),
			msgid.MsgIDS2S_S2SGt2SDisconnect,
			&pb.S2SGt2SDisconnect{
				SesID: s.Id,
				Why:   reason,
			}, 0, 0,
		)

		if s.conn != nil {
			_ = s.conn.Close()
		}

		zap.L().Info("disconnect",
			zap.Inline(s),
			zap.String("reason", reason.String()),
		)
	}
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
	s.loopCount.Store(2)

	thread.GoSafe(func() { s.sendLoop(ctx) })
	thread.GoSafe(func() { s.readLoop(ctx, netCfg) })

	zap.L().Info("connect", zap.Inline(s))
}

func (s *Session) UpdateSerId(gameID int32) {
	s.GameID.Store(gameID)
}

func (s *Session) getSerID(_ pb.Server) uint8 {
	return uint8(s.GameID.Load())
}
