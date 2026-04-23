package msg

import (
	"server/game/role"
	"server/game/role/login_mgr"
	"server/game/role/role_mgr"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"time"

	"google.golang.org/protobuf/proto"
)

func init() {
	router.S().OnG(msgid.MsgIDS2S_S2SReqLogin, onLogin) // 角色登录
	router.S().OnG(msgid.MsgIDS2S_S2SGt2SDisconnect, onDisconnect)

	router.C().On(msgid.MsgIDC2S_C2SHeartBeat, onHeartBeat) // 心跳
}

/*-------------------角色消息-----------------*/
func onHeartBeat(_ gctx.Context, msgIn proto.Message, r *role.Role) {
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
	login_mgr.Mgr.Online(msg)
}

func onDisconnect(_ gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.S2SGt2SDisconnect)
	role_mgr.Mgr.Kick(msg.SesID)
}
