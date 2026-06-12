package clinet

import (
	"crypto/cipher"
	"math/big"
	"net"
	"server/api/pb"
	"server/pkg/crypt/gaes"
	"server/pkg/thread"
	"server/pkg/util"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	sesID           = uint32(1)
	sendPacketLimit = 1024
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
	return util.IToString(s.Id) + "_" + s.Ip.String()
}
func (s *Session) ID() uint32 {
	return s.Id
}

func (s *Session) OnConnect() {
	zap.S().Debugf("%s connect", s.String())
}

func (s *Session) OnClosed() {
	// zap.S().Debugf("%s disconnect", s.String())

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
	defer func() {
		waitGroup.Done()
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
				zap.S().Debugf("session %d close by recv thread exit", s.Id)
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
		zap.S().Warnf("%s gmsg cnt per min[%d] > limit[%d]", s.String(), s.pkgCnt1Min, cfg.RpmLimit)
		s.Close()
	}
	s.pkgCnt1Min = 0
}
