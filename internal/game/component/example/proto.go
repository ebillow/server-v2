package example

import (
	pb2 "server/api/pb"
	msgid2 "server/api/pb/msgid"
	"server/internal/game/role"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"

	"google.golang.org/protobuf/proto"
)

func init() {
	// 客户端消息
	router.C().On(msgid2.MsgIDC2S_C2SEcho, onEchoCli)

	// 服务器消息
	router.S().On(msgid2.MsgIDS2S_S2SNone, onEchoSer)
}

func onEchoCli(_ gctx.Context, msgBase proto.Message, r *role.Role) {
	msg := msgBase.(*pb2.C2SEcho)
	msgOut := pb2.GetS2CEcho()
	msgOut.ID = msg.ID
	msgOut.Name = msg.Name
	msgOut.Level = msg.Level
	msgOut.Exp = msg.Exp
	msgOut.Data = msg.Data
	msgOut.CliTime = msg.Time

	r.Send(msgOut)
	pb2.PutS2CEcho(msgOut)
	pb2.PutC2SEcho(msg)
}

func onEchoSer(_ gctx.Context, msg proto.Message, r *role.Role) {

}
