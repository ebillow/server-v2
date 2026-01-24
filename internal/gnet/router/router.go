package snet

import (
	"errors"
	"sync/atomic"
)

const MsgMaxCount = 65535

var (
	CliMsgRouter = newClientRouter(MsgMaxCount)
	SerMsgRouter = newServerRouter(MsgMaxCount)
	netStart     atomic.Bool
)

var (
	errAPINotFind         = errors.New("api not exist")
	errMsgIDBiggerThanMax = errors.New("msg id bigger than max")
)
