package role

import (
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"server/internal/pb/msgid"
)

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

type ICRouter interface {
	Register(msgID msgid.MsgIDC2S, df func(msg proto.Message, r *Role))
	Handle(msg *nats.Msg, r *Role) error
}

type ISRouter interface {
	Register(msgID msgid.MsgIDS2S, df func(msg proto.Message, r *Role))
	Handle(msg *nats.Msg, r *Role) error
}

// ---------------------------------------------------------
var (
	loginMgr     ILoginMgr
	roleMgr      IRoleMgr
	cliMsgRouter ICRouter
	serMsgRouter ISRouter
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

// CRouter 客户端消息路由---MsgRouter ---------------------------------------------------------
func CRouter() ICRouter {
	return cliMsgRouter
}

func InjectCRouter(rt ICRouter) {
	cliMsgRouter = rt
}
func SRouter() ISRouter {
	return serMsgRouter
}

func InjectSRouter(rt ISRouter) {
	serMsgRouter = rt
}
