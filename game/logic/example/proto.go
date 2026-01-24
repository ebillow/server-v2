package example

import (
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/internal/pb/msgid"
)

func init() {
	//客户端消息
	role.CRouter().Register(msgid.MsgIDC2S_C2SBindAcc, onGetData)

	//服务器消息
	role.SRouter().Register(msgid.MsgIDS2S_S2SNone, onServerMsgNone)
}

func onGetData(msg proto.Message, r *role.Role) {

}

func onServerMsgNone(msg proto.Message, r *role.Role) {

}
