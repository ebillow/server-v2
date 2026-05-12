package gnet

import "server/pkg/idgen"

func GateIDFromSesID(gateID uint64) uint8 {
	return uint8(idgen.MachineID(int64(gateID)))
}
