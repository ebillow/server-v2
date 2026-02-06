package role

import (
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"server/pkg/gnet/gctx"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

type IRoleMgr interface {
	Add(roleID uint64, sesID uint64, r *Role)
	Delete(roleID uint64, sesID uint64)
	KickRoleAndWait(roleID uint64)
	CloseAndWait()
	PostEvent(roleID uint64, evt Event)
	PostEventBySesID(sesID uint64, evt Event)
}

type ILoginMgr interface {
	Online(msg *pb.S2SReqLogin)
	Offline(data *DataToSave)
}

type ICRouter interface {
	RoleMsg(msgID msgid.MsgIDC2S, df func(msg proto.Message, r *Role, c gctx.Context))
	HandleWithRole(natMsg *pb.NatsMsg, raw *nats.Msg, r *Role)
}

type ISRouter interface {
	Msg(msgID msgid.MsgIDS2S, df func(msg proto.Message, c gctx.Context))
	RoleMsg(msgID msgid.MsgIDS2S, df func(msg proto.Message, r *Role, c gctx.Context))
	HandleWithRole(natsMsg *pb.NatsMsg, raw *nats.Msg, r *Role)
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

// cRouter 客户端消息路由---MsgRouter ---------------------------------------------------------
func cRouter() ICRouter {
	return cliMsgRouter
}

func InjectCRouter(rt ICRouter) {
	cliMsgRouter = rt
}
func sRouter() ISRouter {
	return serMsgRouter
}

func InjectSRouter(rt ISRouter) {
	serMsgRouter = rt
}
