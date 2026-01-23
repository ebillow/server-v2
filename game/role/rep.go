package role

import "github.com/nats-io/nats.go"

type IRoleMgr interface {
	Add(roleID uint64, sesID uint64, r *Role)
	Delete(roleID uint64, sesID uint64)
	PostCloseAndWait(roleID uint64)
	CloseAndWait()
	PostEvent(roleID uint64, evt Event)
	PostEventBySesID(sesID uint64, evt Event)
}

type ILoginMgr interface {
	Offline(data *DataToSave)
}

type IRouter interface {
	Handle(msg *nats.Msg, r *Role) error
}

// ---------------------------------------------------------
var (
	loginMgr     ILoginMgr
	roleMgr      IRoleMgr
	cliMsgRouter IRouter
	serMsgRouter IRouter
)

// LoginMgr ---------------------------------------------------------
func LoginMgr() ILoginMgr {
	return loginMgr
}

func InjectLoginMgr(mgr ILoginMgr) {
	loginMgr = mgr
}

// RoleMgr ---------------------------------------------------------
func RoleMgr() IRoleMgr {
	return roleMgr
}

func InjectRoleMgr(mgr IRoleMgr) {
	roleMgr = mgr
}

// MsgRouter ---------------------------------------------------------
func MsgRouter(cli bool) IRouter {
	if cli {
		return cliMsgRouter
	} else {
		return serMsgRouter
	}
}

func InjectMsgRouter(cli IRouter, ser IRouter) {
	cliMsgRouter = cli
	serMsgRouter = ser
}
