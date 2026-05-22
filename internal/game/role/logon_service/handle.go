package logon_service

import (
	"server/api/pb"
	msgid2 "server/api/pb/msgid"
	role2 "server/internal/game/role"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"time"

	"google.golang.org/protobuf/proto"
)

func init() {
	router.S().OnG(msgid2.MsgIDS2S_S2SReqLogin, onLogin) // 角色登录
	router.S().OnG(msgid2.MsgIDS2S_S2SGt2SDisconnect, onDisconnect)

	router.C().On(msgid2.MsgIDC2S_C2SHeartBeat, onHeartBeat) // 心跳
}

/*-------------------角色消息-----------------*/
func onHeartBeat(_ gctx.Context, msgIn proto.Message, r *role2.Role) {
	msg := msgIn.(*pb.C2SHeartBeat)
	now := time.Now()
	r.Send(&pb.S2CHeartBeat{
		CliTime: msg.CliTime,
		SerTime: now.Unix(),
	})
	r.HeartbeatTimeOut = 0
	r.LastHeartbeat = now
}

/*-------------------非角色消息-----------------*/
func onLogin(_ gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.S2SReqLogin)
	Mgr.Login(msg)
}

func onDisconnect(_ gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.S2SGt2SDisconnect)
	role2.Mgr.Kick(msg.SesID)
}
