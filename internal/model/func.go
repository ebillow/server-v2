package model

import "server/internal/pb"

func GetRoleID(accID uint64) uint64 {
	return accID
}

func GetAccID(roleID uint64) uint64 {
	return roleID
}

func GetCompName(comID pb.TypeComp) string {
	return pb.TypeComp_name[int32(comID)]
}
