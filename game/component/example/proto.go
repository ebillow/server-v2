package example

import (
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

func init() {
	// 客户端消息
	router.C().RoleMsg(msgid.MsgIDC2S_C2SEcho, onEchoCli)

	// 服务器消息
	router.S().RoleMsg(msgid.MsgIDS2S_S2SNone, onEchoSer)
}

func onEchoCli(msgBase proto.Message, r *role.Role, _ gctx.Context) {
	msg := msgBase.(*pb.C2SEcho)
	r.Send(&pb.S2CEcho{
		ID:    msg.ID,
		Name:  msg.Name,
		Level: msg.Level,
		Exp:   msg.Exp,
		Data:  msg.Data,
	})
}

func onEchoSer(msg proto.Message, r *role.Role, _ gctx.Context) {

}
