package session

import (
	"crypto/cipher"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"server/pkg/gnet"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"server/pkg/idgen"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"server/pkg/thread"
	"server/pkg/util"
	"time"
)

const (
	sendCacheLen   = 1024
	maxRecvMsgSize = 102400
)

const (
	SesClose = 0x00000001
	SesInit  = 0x00000100
)

type MsgSend struct {
	ID   uint32
	Data []byte
}

// Session 客户端和gate的网络会话
type Session struct {
	Id     uint64
	GameID int32
	conn   *websocket.Conn

	in     chan []byte
	out    chan *MsgSend
	events chan gctx.Context

	pkgCnt     uint32
	pkgCnt1Min int
	Ip         string

	ctrl          chan struct{}
	flag          util.Flag
	disConnReason pb.DisconnectReason

	deCyp cipher.BlockMode
	enCpy cipher.BlockMode
}

func (s *Session) OnConnect() {
	zap.L().Info("connect", zap.Inline(s))
}

func (s *Session) OnClosed() {
	zap.L().Info("disconnect", zap.Inline(s))
	if s.flag.Has(SesInit) {
		RemoveSession(s.Id)
		gnet.SendToGame(s.GameID, &pb.S2SGt2SDisconnect{
			SesID: s.Id,
			Why:   s.disConnReason,
		}, 0, 0)
	}
}

// Close 关闭,非线程安全,只能在消息里调用
func (s *Session) Close(why pb.DisconnectReason) {
	s.disConnReason = why
	s.flag.Add(SesClose)
}

// Init ses初始化
func (s *Session) Init(cs2Key, s2cKey []byte) error {
	s.flag.Add(SesInit)
	s.Id = uint64(idgen.Gen())

	var err error
	// var aesIV = []byte("093po54iuy876tre") // todo
	// if len(cs2Key) > 0 {
	// 	s.deCyp, err = gaes.NewDecrypter(cs2Key, aesIV)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// if len(s2cKey) > 0 {
	// 	s.enCpy, err = gaes.NewEncrypter(s2cKey, aesIV)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	AddSession(s.Id, s)
	return err
}

func (s *Session) Post(msg gctx.Context) {
	select {
	case s.events <- msg:
	default:
		zap.L().Error("[post] events channel full", zap.Inline(s))
	}
}

func (s *Session) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("s.ip", s.Ip)
	encoder.AddUint64("s.id", s.Id)
	return nil
}

// start recv loop
func (s *Session) start() {
	s.in = make(chan []byte) // no active_role
	s.ctrl = make(chan struct{})
	s.out = make(chan *MsgSend, netCfg.OutChanSize)
	s.events = make(chan gctx.Context, netCfg.OutChanSize)

	_ = s.Init([]byte(""), []byte(""))

	go s.sendLoop()
	go s.readLoop(netCfg)

	waitGroup.Add(1)
	s.mainLoop(netCfg)
}

// main
func (s *Session) mainLoop(cfg *Config) {
	defer func() {
		waitGroup.Done()
		if err := recover(); err != nil {
			thread.PrintStack(err)
		}
	}()

	tick := time.NewTicker(time.Minute)

	defer func() {
		s.OnClosed()
		err := s.conn.Close()
		if err != nil {
			zap.L().Warn("close websocket err", zap.Inline(s), zap.Error(err))
		}
		close(s.ctrl)
		tick.Stop()
	}()

	s.OnConnect()

	for {
		select {
		case cliMsg, ok := <-s.in:
			if !ok {
				// zap.S().Debugf("session %d close by recv thread exit", s.Id)
				return
			}

			s.pkgCnt++
			s.pkgCnt1Min++

			s.onRecvClientMsg(cliMsg)
		case c := <-s.events:
			s.onRecvServerMsg(c)
		case <-tick.C:
			s.check1Min(cfg)
		case <-shutDown:
			s.Close(pb.DisconnectReason_ShutDown)
		}

		if s.flag.Has(SesClose) {
			return
		}
	}
}

func (s *Session) check1Min(cfg *Config) {
	if cfg.RpmLimit > 0 && s.pkgCnt1Min > cfg.RpmLimit {
		zap.L().Warn("pkg cnt per min > limit", zap.Inline(s), zap.Int("cnt", s.pkgCnt1Min))
		s.Close(pb.DisconnectReason_Limit)
	}
	s.pkgCnt1Min = 0
}

func (s *Session) getSerID(ser pb.Server) int32 {
	return s.GameID
}

func (s *Session) onRecvClientMsg(src []byte) {
	msgID, seq, data, err := Decode(src, s.deCyp)
	if err != nil {
		zap.L().Warn("read packet err", zap.Inline(s), zap.Error(err))
		s.Close(pb.DisconnectReason_DecodeErr)
		return
	}

	if msgID != uint32(msgid.MsgIDC2S_C2SInit) && !s.flag.Has(SesInit) {
		zap.L().Error("not init", zap.Inline(s))
		s.Close(pb.DisconnectReason_InitErr)
		return
	}
	if seq != 0 && seq != s.pkgCnt {
		zap.L().Error("seq num err:", zap.Inline(s), zap.Uint32("seq", seq), zap.Uint32("cur", s.pkgCnt))
		s.Close(pb.DisconnectReason_PkgCntErr)
		return
	}

	serType := pb.Server(msgID / 100000)
	serID := s.getSerID(serType)
	if serType == pb.Server_Gateway {
		C().Handle(msgID, data, s)
	} else {
		err = msgq.Q.Send(serType, serID, msgID, data, 0, s.Id)
		if err != nil {
			zap.L().Info(">>> to server: "+msgid.MsgIDC2S_name[int32(msgID)],
				zap.Uint32("msgID", msgID),
				zap.String("to", serType.String()),
				zap.Int32("idx", serID),
				zap.Inline(s),
				logger.Magenta.Field(),
			)
		}
		if trace.Rule.ShouldLog(msgID, 0, s.Id) {
			zap.L().Info(">>> to server: "+msgid.MsgIDC2S_name[int32(msgID)],
				zap.Uint32("msgID", msgID),
				zap.String("to", serType.String()),
				zap.Int32("idx", serID),
				zap.Inline(s),
				logger.Magenta.Field(),
			)
		}
	}
}

func (s *Session) onRecvServerMsg(c gctx.Context) {
	if c.Msg.Forward {
		s.SendBytes(c.Msg.MsgID, c.Msg.Data)
	} else {
		S().Handle(c.Msg, c.Raw, s)
	}
}
