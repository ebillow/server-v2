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
	router.C().On(msgid.MsgIDC2S_C2SEcho, onEchoCli)

	// 服务器消息
	router.S().On(msgid.MsgIDS2S_S2SNone, onEchoSer)
}

func onEchoCli(_ gctx.Context, msgBase proto.Message, r *role.Role) {
	msg := msgBase.(*pb.C2SEcho)
	r.Send(&pb.S2CEcho{
		ID:    msg.ID,
		Name:  msg.Name,
		Level: msg.Level,
		Exp:   msg.Exp,
		Data:  msg.Data,
	})
}

func onEchoSer(_ gctx.Context, msg proto.Message, r *role.Role) {

}
