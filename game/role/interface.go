package role

type IRoleMgr interface {
	Add(roleID uint64, sesID uint64, r *Role)
	Delete(roleID uint64, sesID uint64)
	PostCloseAndWait(roleID uint64)
	CloseAndWait()
	PostEvent(roleID uint64, evt Event)
	PostEventBySesID(sesID uint64, evt Event)
}

type DataInDB struct {
	ID   uint64
	Data string
}

type ILoginMgr interface {
	Offline(data *DataInDB)
}

var (
	loginMgr ILoginMgr
	roleMgr  IRoleMgr
)

func GetLoginMgr() ILoginMgr {
	return loginMgr
}

func SetLoginMgr(mgr ILoginMgr) {
	loginMgr = mgr
}

func GetRoleMgr() IRoleMgr {
	return roleMgr
}

func SetRoleMgr(mgr IRoleMgr) {
	roleMgr = mgr
}
