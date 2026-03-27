package router

import (
	"errors"
)

var (
	cliMsgRouter = newClientRouter()
	serMsgRouter = newServerRouter()
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
