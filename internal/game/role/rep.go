package role

import (
	"server/api/pb"
)

type ICompCreate interface {
	Create(r *Role)
}

type ILoginMgr interface {
	Login(msg *pb.S2SReqLogin)
	Logout(data *DataToSave)
	SaveRole(data *DataToSave, saveBoth bool)
}

// ---------------------------------------------------------
var (
	loginMgr   ILoginMgr
	compCreate ICompCreate
)

// LoginMgr ---------------------------------------------------------
func LoginMgr() ILoginMgr {
	return loginMgr
}

func InjectLoginMgr(mgr ILoginMgr) {
	loginMgr = mgr
}

func InjectCompCreate(rt ICompCreate) {
	compCreate = rt
}
