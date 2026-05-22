package role

import (
	"server/api/pb"
	msgid2 "server/api/pb/msgid"
	"server/pkg/gnet/gctx"

	"google.golang.org/protobuf/proto"
)

type ICompCreate interface {
	Create(r *Role)
}

type ILoginMgr interface {
	Login(msg *pb.S2SReqLogin)
	Logout(data *DataToSave)
	SaveRole(data *DataToSave, saveBoth bool)
}

type ICRouter interface {
	On(msgID msgid2.MsgIDC2S, df func(c gctx.Context, msg proto.Message, r *Role))
	Handle(ctx gctx.Context)
}

type ISRouter interface {
	OnG(msgID msgid2.MsgIDS2S, df func(c gctx.Context, msg proto.Message))
	On(msgID msgid2.MsgIDS2S, df func(c gctx.Context, msg proto.Message, r *Role))
	Handle(ctx gctx.Context)
}

// ---------------------------------------------------------
var (
	loginMgr     ILoginMgr
	cliMsgRouter ICRouter
	serMsgRouter ISRouter
	compCreate   ICompCreate
)

// LoginMgr ---------------------------------------------------------
func LoginMgr() ILoginMgr {
	return loginMgr
}

func InjectLoginMgr(mgr ILoginMgr) {
	loginMgr = mgr
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

func InjectCompCreate(rt ICompCreate) {
	compCreate = rt
}
