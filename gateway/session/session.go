package session

import (
	"crypto/cipher"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net"
	"server/pkg/crypt/gaes"
	"server/pkg/flag"
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
	sendPacketLimit = 1024
)

const (
	packetHeadLen = 4
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
	Id   uint64
	conn *websocket.Conn

	in  chan []byte
	out chan *MsgSend

	pkgCnt     uint32
	pkgCnt1Min int
	Ip         net.IP

	ctrl chan struct{}
	flag util.Flag

	deCyp cipher.BlockMode
	enCpy cipher.BlockMode
}

func (s *Session) String() string {
	return util.Uint64ToString(s.ID()) + "_" + s.Ip.String()
}
func (s *Session) ID() uint64 {
	return s.Id
}

func (s *Session) OnConnect() {
	logger.Tracef("%s connect", s.String())
}

func (s *Session) OnClosed() {
	// logger.Tracef("%s disconnect", s.String())

	if s.flag.Has(SesInit) {
		RemoveCliSession(s.Id)
	}
}

// close 关闭,非线程安全,只能在消息里调用
func (s *Session) Close() {
	s.flag.Add(SesClose)
}

// Init ses初始化
func (s *Session) Init(cs2Key, s2cKey []byte) error {
	s.flag.Add(SesInit)
	s.Id = uint64(idgen.Gen())

	var err error
	var aesIV = []byte("093po54iuy876tre") // todo
	if len(cs2Key) > 0 {
		s.deCyp, err = gaes.NewDecrypter(cs2Key, aesIV)
		if err != nil {
			return err
		}
	}
	if len(s2cKey) > 0 {
		s.enCpy, err = gaes.NewEncrypter(s2cKey, aesIV)
		if err != nil {
			return err
		}
	}
	AddCliSession(s.Id, s)
	return err
}

// start recv loop
func (s *Session) start() {
	s.in = make(chan []byte) // no active_role
	s.ctrl = make(chan struct{})
	s.out = make(chan *MsgSend, netCfg.OutChanSize)

	go s.sendLoop()
	go s.readLoop(netCfg)

	waitGroup.Add(1)
	s.mainLoop(netCfg)
}

// main
func (s *Session) mainLoop(cfg *Config) {
	logger.Debugf("%s cli main loop start", s.String())
	defer func() {
		waitGroup.Done()
		logger.Debugf("%s cli main loop stop", s.String())

		if err := recover(); err != nil {
			thread.PrintStack(err)
		}
	}()

	tick := time.NewTicker(time.Minute)

	defer func() {
		s.OnClosed()
		err := s.conn.Close()
		if err != nil {
			logger.Warnf("close websocket %s err:%v", s.String(), err)
		}
		logger.Debugf("close websocket %s", s.String())
		close(s.ctrl)
		tick.Stop()
	}()

	s.OnConnect()

	for {
		select {
		case cliMsg, ok := <-s.in:
			if !ok {
				// logger.Debugf("session %d close by recv thread exit", s.Id)
				return
			}

			s.pkgCnt++
			s.pkgCnt1Min++

			s.onRecvCliMsg(cliMsg)
		case <-tick.C:
			s.check1Min(cfg)
		case <-shutDown:
			s.Close()
		}

		if s.flag.Has(SesClose) {
			return
		}
	}
}

func (s *Session) check1Min(cfg *Config) {
	if cfg.RpmLimit > 0 && s.pkgCnt1Min > cfg.RpmLimit {
		logger.Warnf("%s pkg cnt per min[%d] > limit[%d]", s.String(), s.pkgCnt1Min, cfg.RpmLimit)
		s.Close()
	}
	s.pkgCnt1Min = 0
}

func (s *Session) getSerID(ser pb.Server) int32 {
	return 0
}

func (s *Session) onRecvCliMsg(src []byte) {
	msgID, seq, data, err := Decode(src, s.deCyp)
	if err != nil {
		logger.Warnf("%s read packet err:%v", s.String(), err)
		s.Close()
		return
	}

	if msgID != uint32(msgid.MsgIDC2S_C2SInit) && !s.flag.Has(SesInit) {
		logger.Errorf("%s not init", s.String())
		s.Close()
		return
	}
	if seq != 0 && seq != s.pkgCnt {
		logger.Errorf("%s sequeue num err: %d should be %d", s.String(), seq, s.pkgCnt)
		logger.Debug("data=%v", data)
		s.Close()
		return
	}

	serType := pb.Server(msgID / 100000)
	serID := s.getSerID(serType)
	if serType == pb.Server_Gateway {
		err = cliMsgHandler.Handle(msgID, data, s)
		if err != nil {
			logger.Warnf("%s recv invalid msg from cli [%d]", s.String(), msgID)
			s.Close()
		}
	} else {
		serName := flag.SrvName(serType)
		msgq.Q.Send(serName, serID, msgID, data, 0, s.Id)

		if trace.Rule.ShouldLog(msgID, 0, s.Id) {
			zap.L().Info(">>> to server: "+msgid.MsgIDC2S_name[int32(msgID)],
				zap.Uint32("msgID", msgID),
				zap.String("to", serName),
				zap.Int32("idx", serID),
				zap.Uint64("sessID", s.Id),
				logger.Magenta.Field(),
			)
		}
	}
}
