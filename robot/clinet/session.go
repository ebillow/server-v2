package clinet

import (
	"crypto/cipher"
	"github.com/gorilla/websocket"
	"math/big"
	"net"
	"server/pkg/crypt/gaes"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/thread"
	"server/pkg/util"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"
)

var (
	sesID           = uint32(1)
	sendPacketLimit = 1024
)

const (
	packetHeadLen = 4
)

const (
	SesClose      = 0x00000001
	FightClose    = 0x00000010
	SesInit       = 0x00000100
	FightStreamOk = 0x00001000
)

type IUnit interface {
	SecLoop()
	OnDisconnect()
}

// Session 客户端和gate的网络会话
type Session struct {
	Id   uint32
	conn *websocket.Conn

	in     chan []byte
	out    chan *pkgWriter
	evt    chan Evt
	serMsg chan *pb.SrvMsg

	pkgCnt     uint32
	pkgCnt1Min int
	Ip         net.IP
	pkgSend    uint32

	ctrl     chan struct{}
	fightDie chan struct{}
	flag     util.Flag

	DeCyp cipher.BlockMode
	EnCpy cipher.BlockMode

	U IUnit

	S2cPrivate *big.Int
	C2sPrivate *big.Int
}

func (s *Session) String() string {
	return util.Uint32ToString(s.ID()) + "_" + s.Ip.String()
}
func (s *Session) ID() uint32 {
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
	s.U.OnDisconnect()
}

// close 关闭,非线程安全,只能在消息里调用
func (s *Session) Close() {
	s.flag.Add(SesClose)
}

// closeToFt 关闭到fight的网络会话
func (s *Session) CloseToFt() {
	s.flag.Add(FightClose)
}

func (s *Session) FtStreamSuccess() {
	s.flag.Add(FightStreamOk)
}

func (s *Session) Init(cs2Key, s2cKey []byte) error {
	var err error
	var aesIV = []byte("093po54iuy876tre") // todo
	if len(cs2Key) > 0 {
		s.EnCpy, err = gaes.NewDecrypter(cs2Key, aesIV)
		if err != nil {
			return err
		}
	}
	if len(s2cKey) > 0 {
		s.DeCyp, err = gaes.NewEncrypter(s2cKey, aesIV)
		if err != nil {
			return err
		}
	}
	s.flag.Add(SesInit)
	s.Id = atomic.AddUint32(&sesID, 1)
	AddCliSession(s.Id, s)

	return err
}

// start recv loop
func (s *Session) start(cfg *Config) {
	s.evt = make(chan Evt, 100)
	s.out = make(chan *pkgWriter, cfg.OutChanSize)

	go s.sendLoop(cfg)

	waitGroup.Add(1)
	go s.mainLoop(cfg)

	s.recvLoop(cfg)
}

// main
func (s *Session) mainLoop(cfg *Config) {
	logger.Debug("main loop start")
	defer func() {
		waitGroup.Done()
		logger.Debugf("main loop stop %v", waitGroup)

		if err := recover(); err != nil {
			thread.PrintStack(err)
		}
	}()

	s.serMsg = make(chan *pb.SrvMsg, cfg.EvtChanSize)
	tick := time.NewTicker(time.Minute)
	tSec := time.NewTicker(time.Second)

	defer func() {
		s.OnClosed()
		close(s.ctrl)
		tick.Stop()
		tSec.Stop()
	}()

	s.OnConnect()

	for {
		select {
		case cliMsg, ok := <-s.in:
			if !ok {
				logger.Debugf("session %d close by recv thread exit", s.Id)
				return
			}

			s.pkgCnt++
			s.pkgCnt1Min++

			s.onRecvCliMsg(cliMsg)
		case e := <-s.evt:
			s.onEvent(e)
		case <-tSec.C:
			if s.U != nil {
				s.U.SecLoop()
			}
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

func (s *Session) DebugForwardToGame(msgId uint16, message proto.Message) {
	if message != nil {
		data, err := proto.Marshal(message)
		if err == nil {
			s.forwardToGame(msgId, data)
		}
	} else {
		s.forwardToGame(msgId, nil)
	}
}

// forwardToGame	转发数据到game
func (s *Session) forwardToGame(msgId uint16, msgData []byte) {
	// msg := &pb.SrvMsg{
	//	ID:  uint32(msgId),
	//	Msg: msgData,
	// }
	// if s.StreamGm == nil {
	//	logger.Errorf("%s stream to game no open %d", s.String(), msgId)
	//	s.Close()
	//	return
	// }
	//
	// if err := s.StreamGm.SendPB(msg); err != nil {
	//	logger.Errorf("forward to game:%v", err)
	//	s.Close()
	//	return
	// }
	// if msgId != uint16(pb.MsgIDC2S_C2SHeartBeat) {
	//	logger.Debugf("%s forward to game:[%d]%s", s.String(), msgId, pb.MsgIDC2S_name[int32(msgId)])
	// }
}

func (s *Session) forwardToFight(msgID uint16, msgData []byte) {
	// msg := &pb.SrvMsg{ID: uint32(msgID), Msg: msgData}
	// if s.StreamFt == nil {
	//	logger.Errorf("stream to ft no open %d", msgID)
	//	s.Close()
	//	return
	// }
	//
	// if err := s.StreamFt.SendPB(msg); err != nil {
	//	logger.Errorf("forward to fight:%v", err)
	//	s.Close()
	//	return
	// }
	//
}

// startStreamGm 开启到game的流
// func (s *Session) startStreamGm(gameID uint32, acc uint64) {
// 	if s.StreamGm != nil {
// 		logger.Error("can not create double stream to game")
// 		return
// 	}
// 	// 连接到已选定game服务器
// 	//	logger.Debugf("%s get service", acc)
// 	// conn := gnet.Get("game", gameID)
// 	// if conn == nil {
// 	//	logger.Errorf("cannot get game service, id:%d", gameID)
// 	//	s.Close()
// 	//	return
// 	// }
// 	//
// 	// //	logger.Debugf("%s new client", acc)
// 	// cli := pb.NewSrvServiceClient(conn)
// 	// // 开启到游戏服的流
// 	// mtdata := metadata.New(map[string]string{"acc": util.ToString(acc)})
// 	// ctx := metadata.NewOutgoingContext(context.Background(), mtdata)
// 	// //	logger.Debugf("%s start srv", acc)
// 	// stream, err := cli.SrvSrv(ctx)
// 	// if err != nil {
// 	//	logger.Errorf("%d start game stream[%s] err:%v", acc, conn.Router(), err)
// 	//	s.Close()
// 	//	return
// 	// }
// 	//
// 	// s.StreamGm = stream
// 	// logger.Debugf("%s acc=%d start to game%d stream success", s.String(), acc, gameID)
// 	// // 读取GAME返回消息的goroutine
// 	// go func(s *Session, stream pb.SrvService_SrvSrvClient, desc string) {
// 	//	for {
// 	//		in, err := stream.Recv()
// 	//		if err == io.EOF { // 流关闭
// 	//			logger.Debugf("%s game stream close %v", desc, err)
// 	//			return
// 	//		}
// 	//		if err != nil {
// 	//			logger.Errorf("%s game stream close %v", desc, err)
// 	//			return
// 	//		}
// 	//		select {
// 	//		case s.serMsg <- in:
// 	//		case <-s.ctrl:
// 	//			logger.Debugf("%s game stream close by main goroutine", desc)
// 	//			return
// 	//		}
// 	//	}
// 	// }(s, stream, s.String())
// }
//
// func (s *Session) startStreamFt(roleGuid int, ftID uint32, btGuid int, token string) {
// 	// logger.Debugf("%d start stream ft %d", roleGuid, ftID)
// 	if s.StreamFt != nil {
// 		logger.Error("can not create double stream to ft")
// 		return
// 	}
// 	// conn := gnet.Get("fight", ftID)
// 	// if conn == nil {
// 	//	logger.Errorf("cannot get fight service, id:%d", ftID)
// 	//	s.Close()
// 	//	return
// 	// }
// 	//
// 	// cli := pb.NewSrvServiceClient(conn)
// 	// // 开启到fight服的流
// 	// mtdata := metadata.New(map[string]string{"guid": strconv.Itoa(roleGuid), "region": strconv.Itoa(btGuid), "token": token})
// 	// ctx := metadata.NewOutgoingContext(context.Background(), mtdata)
// 	// stream, err := cli.SrvSrv(ctx)
// 	// if err != nil {
// 	//	logger.Errorf("%s start fight stream %s err:%v", s.String(), conn.Router(), err)
// 	//	s.Close()
// 	//	return
// 	// }
// 	//
// 	// s.flag.Del(FightClose)
// 	// s.fightDie = make(chan struct{})
// 	// s.StreamFt = stream
// 	// s.flag.Del(FightStreamOk)
// 	//
// 	// // 读取fight返回消息的goroutine
// 	// go func(sess *Session, stream pb.SrvService_SrvSrvClient, desc string) {
// 	//	for {
// 	//		in, err := stream.Recv()
// 	//		if err == io.EOF { // 流关闭
// 	//			logger.Debugf("%s fight stream close %v", desc, err)
// 	//			return
// 	//		}
// 	//		if err != nil {
// 	//			logger.Errorf("%s fight stream close %v", desc, err)
// 	//			return
// 	//		}
// 	//		select {
// 	//		case sess.serMsg <- in:
// 	//		case <-sess.fightDie:
// 	//			logger.Debugf("%s fight stream close by main goroutine", desc)
// 	//			return
// 	//		}
// 	//	}
// 	// }(s, stream, s.String())
// }
//
// func (s *Session) ReConn(msg *pb2.MsgReConn) {
// 	s.startStreamGm(msg.GameID, msg.Acc)
// 	b, err := proto.Marshal(msg)
// 	if err != nil {
// 		logger.Warnf("ReConn marshal err:%v", err)
// 		s.Close()
// 	}
// 	s.forwardToGame(uint16(pb.MsgIDC2S_C2SReConn), b)
// }
