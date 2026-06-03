package role

import (
	"server/api/pb"
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

type IRouter interface {
	Register(msgID uint32, cf func() pb.VTMessage, usePool bool, df func(c gctx.Context, msg proto.Message)) error
	Handle(c gctx.Context) error
}

// ---------------------------------------------------------
var (
	loginMgr     ILoginMgr
	cliMsgRouter IRouter
	serMsgRouter IRouter
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
func cRouter() IRouter {
	return cliMsgRouter
}

func InjectCRouter(rt IRouter) {
	cliMsgRouter = rt
}
func sRouter() IRouter {
	return serMsgRouter
}

func InjectSRouter(rt IRouter) {
	serMsgRouter = rt
}

func InjectCompCreate(rt ICompCreate) {
	compCreate = rt
}
