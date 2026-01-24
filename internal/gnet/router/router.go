package snet

import (
	"errors"
	"sync/atomic"
)

const MsgMaxCount = 65535

var (
	CliMsgRouter = newClientRouter(65536)
	SerMsgRouter = newServerRouter(65536)
	netStart     atomic.Bool
)

var (
	errAPINotFind      = errors.New("api not find")
	errMsgIDBigThanMax = errors.New("msg id bigger than max")
)
