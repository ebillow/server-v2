package role

type ICompFactory interface {
	Create(r *Role)
}

type ILoginMgr interface {
	Logout(data *DataToSave)
	SaveRole(data *DataToSave, saveBoth bool) bool
}

// ---------------------------------------------------------
var (
	loginMgr    ILoginMgr
	compFactory ICompFactory
)

// LoginMgr ---------------------------------------------------------
func LoginMgr() ILoginMgr {
	return loginMgr
}

func InjectLoginMgr(mgr ILoginMgr) {
	loginMgr = mgr
}
func CompFactory() ICompFactory { return compFactory }

func InjectCompCreate(rt ICompFactory) {
	compFactory = rt
}
