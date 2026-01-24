package gnet

import "server/internal/pb"

func SrvName(serType pb.Server) string {
	return pb.Server_name[int32(serType)]
}

func SrvRoleName(serType pb.Server) string {
	return SrvName(serType) + "_role"
}

func GateIDFromSesID(gateID uint64) int32 {
	return int32(gateID) //todo
}
