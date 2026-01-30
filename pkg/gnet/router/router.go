package router

import (
	"errors"
	"github.com/nats-io/nats.go"
	"sync/atomic"
)

const MsgMaxCount = 655350

var (
	cliMsgRouter = newClientRouter(MsgMaxCount)
	serMsgRouter = newServerRouter(MsgMaxCount)
	netStart     atomic.Bool
)

var (
	errAPINotFind         = errors.New("api not exist")
	errMsgIDBiggerThanMax = errors.New("msg id bigger than max")
)

type Unity interface{}
type Context struct {
	U   Unity
	Msg *nats.Msg
}

func S() *ServerRouter {
	return serMsgRouter
}

func C() *ClientRouter {
	return cliMsgRouter
}

func Start() {
	netStart.Store(true)
}
