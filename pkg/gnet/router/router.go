package router

import (
	"server/pkg/pb/msgid"
)

var (
	cliMsgRouter = newClientRouter(int32(msgid.MsgIDMax_C2SMax))
	serMsgRouter = newServerRouter(int32(msgid.MsgIDMax_S2SMax))
)

func S() *ServerRouter {
	return serMsgRouter
}

func C() *ClientRouter {
	return cliMsgRouter
}
