package robot

import (
	"fmt"
	pb "server/api/pb"
	"server/api/pb/msgid"
	clinet2 "server/internal/robot/clinet"
	"server/pkg/util"
	"time"

	"go.uber.org/zap"

	"google.golang.org/protobuf/proto"
)

type State int

const (
	Init = iota
	InGame
	Disconnect
	Pending
)

type Robot struct {
	s           *clinet2.Session
	id          uint32
	state       State
	stateTime   time.Time
	acc         string
	area        uint32
	Data        *pb.RoleData
	ReconnToken uint32
	gameId      uint32

	lastActTime time.Time

	chats  map[uint64]bool
	chatID uint32

	taskMgr *TimeEvter
}

var cfg = &clinet2.Config{
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

	s, err := clinet2.DailWebsocket(Setup.ServerAddr, cfg)
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
}

func (r *Robot) SecLoop(now time.Time) {
	switch r.state {
	case Init:
		err := r.s.Init(nil, nil)
		if err != nil {
			zap.L().Error("Init fail", zap.Error(err))
			return
		}
		r.Login()
		r.state = Pending
	case InGame:
		TaskRun(r)
	default:
	}
}

func (r *Robot) AddTask(period int64, cb func(*Robot)) {
	r.taskMgr.Add(period, cb)
}

func (r *Robot) OnDisconnect() {
	if r.s != nil {
		r.s.U = nil
	}
	r.s = nil
	r.stateTime = time.Now()
	r.state = Disconnect
	go func() {
		span := util.RandRange(time.Duration(3), 300)
		<-time.After(time.Second * span)
		s, err := clinet2.DailWebsocket(Setup.ServerAddr, cfg)
		if err != nil {
			return
		}

		if r.s != nil {
			r.s.Close()
			r.s = nil
		}
		r.s = s
		s.U = r
		zap.S().Infof("%s reconnect", r.acc)
		r.state = Init
	}()
}

func (r *Robot) GetData() *pb.RoleData {
	return r.Data
}

func (r *Robot) Send(msgId msgid.MsgIDC2S, msg proto.Message) {
	if r.s != nil {
		r.s.SendPB(msgId, msg)
	}
}

func (r *Robot) Login() {
	// reConn := r.Data != nil
	msg := pb.C2SLogin{
		Account: r.acc,
		Dev:     r.acc,
		SdkType: pb.SdkType_Guest,
		Channel: 0,
		// Reconnect: reConn,

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

	// if !slices.Contains(msg.ConnectAcc, r.acc) {
	// 	panic("connect acc not exist")
	// }
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
	r.Send(msgid.MsgIDC2S_C2SHeartBeat, &pb.C2SHeartBeat{
		CliTime: now.UnixMilli(),
	})
}
