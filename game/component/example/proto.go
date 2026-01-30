package example

import (
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/pkg/gnet/router"
	"server/pkg/pb/msgid"
)

func init() {
	// 客户端消息
	router.C().RoleMsg(msgid.MsgIDC2S_C2SBindAcc, onGetData)

	// 服务器消息
	router.S().RoleMsg(msgid.MsgIDS2S_S2SNone, onServerMsgNone)
}

func onGetData(msg proto.Message, r *role.Role) {

}

func onServerMsgNone(msg proto.Message, r *role.Role, qm *nats.Msg) {

}
