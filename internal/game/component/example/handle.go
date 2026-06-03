package example

import (
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/game/role"
	"server/pkg/gnet/gctx"
)

func init() {
	// 客户端消息
	role.OnC2SUsePool(onEchoCli)

	// 服务器消息
	role.OnS2SRoleUsePool(onEchoSer)
}

func onEchoCli(_ gctx.Context, msg *pb.C2SEcho, r *role.Role) {
	msgOut := pb.GetS2CEcho()
	msgOut.ID = msg.ID
	msgOut.Name = msg.Name
	msgOut.Level = msg.Level
	msgOut.Exp = msg.Exp
	msgOut.Data = msg.Data
	msgOut.CliTime = msg.Time

	role.Send(r, msgid.MsgIDS2C_S2CEcho, msgOut)
	pb.PutS2CEcho(msgOut)
	pb.PutC2SEcho(msg)
}

func onEchoSer(_ gctx.Context, msg *pb.S2SEcho, r *role.Role) {

}
