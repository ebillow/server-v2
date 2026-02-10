package robot

import (
	"fmt"
	"go.uber.org/zap"
	"server/pkg/crypt/dh"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"server/robot/clinet"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"
)

type State int

const (
	Init = iota
	InGame
	ReConn
)

type Robot struct {
	s           *clinet.Session
	id          uint32
	state       State
	stateTime   time.Time
	acc         string
	area        uint32
	Data        *pb.RoleData
	ReconnToken uint32
	gameId      uint32

	lastActTime  time.Time
	isDisconnect uint32

	chats  map[uint64]bool
	chatID uint32

	taskMgr *TimeEvter
}

var cfg = &clinet.Config{
	OutChanSize:         1250,
	ReadSocketBuffSize:  32767,
	WriteSocketBuffSize: 32767,
	RpmLimit:            0,
	RecvPkgLenLimit:     1024000,
	EvtChanSize:         100,
}

func NewUnitRobot(id int, area uint32) {
	r := &Robot{
		chats: make(map[uint64]bool),
		area:  area,
	}

	r.id = uint32(id)
	r.acc = fmt.Sprintf("Robot%d", r.id)

	s, err := clinet.DailWebsocket(Setup.ServerAddr, cfg)
	if err != nil {
		return
	}

	InitTask(r)
	r.s = s
	if r.s == nil {
		zap.S().Error("init robot ses nil")
		return
	}
	r.s.U = r
	go checkReconn(r)
}

func (r *Robot) SecLoop() {
	// zap.S().Debugf("secloop")

	switch r.state {
	case Init:
		if time.Now().Sub(r.stateTime) > time.Second*10 {
			r.SendInitMsg()
			r.stateTime = time.Now()
		}
	case ReConn:
		if time.Now().Sub(r.stateTime) > time.Second*10 {
			r.SendInitMsg()
			r.stateTime = time.Now()
		}
	case InGame:
		TaskRun(r)
	}
}

func (r *Robot) AddTask(period int64, cb func(*Robot)) {
	r.taskMgr.Add(period, cb)
}

func checkReconn(r *Robot) {
	t := time.NewTicker(time.Second * 60)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if atomic.LoadUint32(&r.isDisconnect) == 1 {
				s, err := clinet.DailWebsocket(Setup.ServerAddr, cfg)
				if err != nil {
					return
				}
				atomic.StoreUint32(&r.isDisconnect, 0)
				if r.s != nil {
					r.s.Close()
					r.s = nil
				}
				r.s = s
				s.U = r
				zap.S().Infof("%s start reconnect", r.acc)
				if r.Data != nil {
					r.state = ReConn
				} else {
					r.state = Init
				}
			}
		}
	}
}

func (r *Robot) OnDisconnect() {
	if r.s != nil {
		r.s.U = nil
	}
	r.s = nil
	atomic.StoreUint32(&r.isDisconnect, 1)
}

func (r *Robot) GetData() *pb.RoleData {
	return r.Data
}

func (r *Robot) Send(msgId msgid.MsgIDC2S, msg proto.Message) {
	if r.s != nil {
		r.s.SendPB(msgId, msg)
	}
}

func (r *Robot) SendInitMsg() {
	c2SPrivateKey, c2sPublicKey := dh.Exchange()
	r.s.C2sPrivate = c2SPrivateKey

	s2cPrivateKey, s2cPublicKey := dh.Exchange()
	r.s.S2cPrivate = s2cPrivateKey
	msg := &pb.C2SInit{
		S2CPublic: s2cPublicKey.String(),
		C2SPublic: c2sPublicKey.String(),
	}
	r.Send(msgid.MsgIDC2S_C2SInit, msg)
	// zap.S().Infof("%s %s send init msg", r.acc, r.s.String())
}

func (r *Robot) Login() {
	msg := pb.C2SLogin{
		Account: r.acc,
		Dev:     r.acc,
		SdkNo:   pb.ESdkNumber_Guest,
		Channel: 0,

		CliInfo: &pb.ClientInfo{
			DevID: "robot test",
		},
	}
	r.Send(msgid.MsgIDC2S_C2SLogin, &msg)
}

func (r *Robot) ReConn() {
	msg := pb.C2SLogin{
		Account:   r.acc,
		Dev:       r.acc,
		SdkNo:     pb.ESdkNumber_Guest,
		Channel:   0,
		Reconnect: true,

		CliInfo: &pb.ClientInfo{
			DevID: "robot test",
		},
	}
	r.Send(msgid.MsgIDC2S_C2SLogin, &msg)
}

func (r *Robot) initData(msg *pb.RoleData) {
	r.Data = msg
	if r.Data.Items == nil {
		r.Data.Items = make(map[string]int64)
	}
}

func (r *Robot) onLoginSuccess(msg *pb.S2CLogin) {
	r.initData(msg.Player)
	r.gameId = msg.GameID

	// if r.acc != r.Data.Acc {
	// 	zap.S().Warnf("acc err:%s!=%s", r.acc, r.Data.Uid)
	// 	return
	// }

	if Setup.LoginOnly {
		Robots.Store(r.id, true)
		zap.S().Infof("%s login success in world %d", r.Data.Name, r.area)
		return
	}

	r.s.U = r
	r.ReconnToken = msg.Token
	// r.Send(pb.MsgIDC2S_C2SCilentReady, nil)
	// worldId := share.GetWorldFromGuid(r.Data.Guid)
	zap.S().Infof("%s %s %d %s login into success", r.acc, r.Data.Name, r.Data.ID, r.s.String())
	r.state = InGame
	Active(r.Data.ID)
}

func (r *Robot) IsLoginSuccess() bool {
	return r.Data != nil
}

func (r *Robot) heartBeat(now time.Time) {
	// r.SendPB(pb.MsgIDC2S_C2SHeartBeat, &pb.MsgHeartBeat{
	//	CliTime: util.GetNowTimeM(),
	// })
}
