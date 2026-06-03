package logon_service

import (
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/game/role"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"time"
)

func init() {
	router.On(onLogin) // 角色登录
	router.On(onDisconnect)

	role.OnP(onHeartBeat) // 心跳
}

/*-------------------角色消息-----------------*/
func onHeartBeat(_ gctx.Context, msg *pb.C2SHeartBeat, r *role.Role) {
	now := time.Now()
	role.Send(r, msgid.MsgIDS2C_S2CHeartBeat, &pb.S2CHeartBeat{
		CliTime: msg.CliTime,
		SerTime: now.Unix(),
	})
	r.HeartbeatTimeOut = 0
	r.LastHeartbeat = now
}

/*-------------------非角色消息-----------------*/
func onLogin(_ gctx.Context, msg *pb.S2SReqLogin) {
	Mgr.Login(msg)
}

func onDisconnect(_ gctx.Context, msg *pb.S2SGt2SDisconnect) {
	role.Mgr.Kick(msg.SesID)
}
