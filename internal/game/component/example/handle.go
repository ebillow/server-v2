package example

import (
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/game/role"
	"server/pkg/gnet/pkg"
	"server/pkg/gnet/pool"
	"server/pkg/gnet/router"

	"google.golang.org/protobuf/proto"
)

func init() {
	role.OnP(onEchoCli)  // Handle不带role,使用对象池
	router.On(onEchoSer) // Handle不带role，不使用对象池

	role.OnRpc(onEchoWithRoleRpc) // rpc Handle带role，不使用对象池
	router.OnRpcP(onEchoRpc)      // rpc Handle不带role, 使用对象池
}

var (
	echoPool = pool.NewMsgPool[pb.S2CEcho]()
)

func onEchoCli(_ pkg.Head, msg *pb.C2SEcho, r *role.Role) {
	msgOut := echoPool.Get()
	msgOut.ID = msg.ID
	msgOut.Name = msg.Name
	msgOut.Level = msg.Level
	msgOut.Exp = msg.Exp
	msgOut.Data = msg.Data
	msgOut.CliTime = msg.Time

	role.Send(r, msgid.MsgIDS2C_S2CEcho, msgOut)
	echoPool.Put(msgOut)
}

func onEchoSer(_ pkg.Head, msg *pb.S2SEcho) {

}

func onEchoRpc(_ pkg.Head, req *pb.S2SRpcEchoReq, res *pb.S2SRpcEchoRes) {
	res = proto.Clone(req).(*pb.S2SRpcEchoRes)
}

func onEchoWithRoleRpc(_ pkg.Head, req *pb.S2SRpcEchoRoleReq, res *pb.S2SRpcEchoRoleRes, r *role.Role) {
	res = proto.Clone(req).(*pb.S2SRpcEchoRoleRes)
}
