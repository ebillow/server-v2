package example

import (
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/game/role"
	"server/pkg/gnet/router"
)

func init() {
	role.OnP(onEchoCli)  // Handle不带role,使用对象池
	router.On(onEchoSer) // Handle不带role，不使用对象池

	role.OnRpc(onEchoWithRoleRpc) // rpc Handle带role，不使用对象池
	router.OnRpcP(onEchoRpc)      // rpc Handle不带role, 使用对象池
}

func onEchoCli(req *pb.C2SEcho, r *role.Role) {
	res := pb.PoolS2CEcho.Get()
	res.ID = req.ID
	res.Name = req.Name
	res.Level = req.Level
	res.Exp = req.Exp
	res.Data = req.Data
	res.CliTime = req.Time

	role.Send(r, msgid.MsgIDS2C_S2CEcho, res)
	pb.PoolS2CEcho.Put(res)
}

func onEchoSer(req *pb.S2SEcho) {

}

func onEchoRpc(req *pb.S2SRpcEchoReq, res *pb.S2SRpcEchoRes) {
	res.ID = req.ID
	res.Name = req.Name
	res.Level = req.Level
	res.Exp = req.Exp
	res.Data = req.Data
}

func onEchoWithRoleRpc(req *pb.S2SRpcEchoRoleReq, res *pb.S2SRpcEchoRoleRes, r *role.Role) {
	res.ID = req.ID
	res.Name = req.Name
	res.Level = req.Level
	res.Exp = req.Exp
	res.Data = req.Data
}
