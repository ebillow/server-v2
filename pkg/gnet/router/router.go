package router

import (
	"errors"
	"sync/atomic"
)

var (
	cliMsgRouter = newClientRouter()
	serMsgRouter = newServerRouter()
	netStart     atomic.Bool
)

var (
	errAPINotFind         = errors.New("api not exist")
	errMsgIDBiggerThanMax = errors.New("msg id bigger than max")
)

func S() *ServerRouter {
	return serMsgRouter
}

func C() *ClientRouter {
	return cliMsgRouter
}

func Start() {
	netStart.Store(true)
}
