package example

import (
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/game/role"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"

	"google.golang.org/protobuf/proto"
)

func init() {
	role.OnP(onEchoCli)
	role.On(onEchoSer)

	router.OnRpc(onEchoRpc)
}

func onEchoCli(_ gctx.Context, msg *pb.C2SEcho, r *role.Role) {
	msgOut := pb.S2CEcho{}
	msgOut.ID = msg.ID
	msgOut.Name = msg.Name
	msgOut.Level = msg.Level
	msgOut.Exp = msg.Exp
	msgOut.Data = msg.Data
	msgOut.CliTime = msg.Time

	role.Send(r, msgid.MsgIDS2C_S2CEcho, &msgOut)
}

func onEchoSer(_ gctx.Context, msg *pb.S2SEcho, r *role.Role) {

}

func onEchoRpc(_ gctx.Context, req *pb.S2SRpcEchoReq, res *pb.S2SRpcEchoRes) {
	res = proto.Clone(req).(*pb.S2SRpcEchoRes)
}
