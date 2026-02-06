package gnet

import "server/pkg/idgen"

func GateIDFromSesID(gateID uint64) int32 {
	return int32(idgen.ExtractServerId(int64(gateID)))
}
