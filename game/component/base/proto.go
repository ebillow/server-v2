package base

import (
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/game/role/login_mgr"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"time"
)

func init() {
	router.S().Msg(msgid.MsgIDS2S_S2SReqLogin, onLogin)          // 角色登录
	router.C().RoleMsg(msgid.MsgIDC2S_C2SHeartBeat, onHeartBeat) // 心跳
}

/*-------------------角色消息-----------------*/
func onHeartBeat(msgIn proto.Message, r *role.Role) {
	msg := msgIn.(*pb.C2SHeartBeat)
	r.Send(&pb.S2CHeartBeat{
		CliTime: msg.CliTime,
		SerTime: time.Now().Unix(),
	})
}

/*-------------------非角色消息-----------------*/
func onLogin(msgBase proto.Message, qm *nats.Msg) {
	msg := msgBase.(*pb.S2SReqLogin)
	login_mgr.Mgr.Online(msg)
}
