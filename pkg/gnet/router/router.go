package router

import (
	"server/api/pb"
	"server/pkg/flag"
	"server/pkg/gerror"
	"server/pkg/gnet/gctx"
	"sync"
)

// MsgRouter 消息处理器
type MsgRouter struct {
	handlers []IHandler
}

// NewMsgRouter createRoute
func NewMsgRouter(max int32) *MsgRouter {
	r := &MsgRouter{
		make([]IHandler, max),
	}
	return r
}

// Register 注册消息
func (rt *MsgRouter) Register(msgID uint32, cf func() pb.VTMessage, usePool bool, df func(c gctx.Context, msg pb.VTMessage)) error {
	if flag.IsReady() {
		return gerror.New("must register before action")
	}
	if msgID >= uint32(len(rt.handlers)) {
		return gerror.Newf("msg id[%d] out of range", msgID)
	}

	if rt.handlers[msgID] != nil {
		return gerror.Newf("msg id[%d] already register", msgID)
	}

	node := &MsgHandler{
		HandleFunc: df,
	}

	if usePool {
		node.factory = &poolFactory{
			pool: sync.Pool{
				New: func() any { return cf() },
			},
		}
	} else {
		node.factory = &newFactory{create: cf}
	}
	rt.handlers[msgID] = node

	return nil
}

func (rt *MsgRouter) RegisterRpc(msgID uint32, createReq func() pb.VTMessage, createRes func() pb.VTMessage, usePool bool, df func(c gctx.Context, req pb.VTMessage, res pb.VTMessage)) error {
	if flag.IsReady() {
		return gerror.New("must register before action")
	}
	if msgID >= uint32(len(rt.handlers)) {
		return gerror.Newf("msg id[%d] out of range", msgID)
	}

	if rt.handlers[msgID] != nil {
		return gerror.Newf("msg id[%d] already register", msgID)
	}

	node := &RpcHandler{
		HandleFunc: df,
	}

	if usePool {
		node.reqCreate = &poolFactory{
			pool: sync.Pool{
				New: func() any { return createReq() },
			},
		}
		node.resCreate = &poolFactory{
			pool: sync.Pool{
				New: func() any { return createRes() },
			},
		}
	} else {
		node.reqCreate = &newFactory{create: createReq}
		node.resCreate = &newFactory{create: createRes}
	}
	rt.handlers[msgID] = node

	return nil
}

func (rt *MsgRouter) Handle(c gctx.Context) error {
	handler, err := rt.getHandler(c.MsgID)
	if err != nil {
		return err
	}
	return handler.Handle(c)
}

func (rt *MsgRouter) getHandler(id uint32) (n IHandler, err error) {
	if id >= uint32(len(rt.handlers)) {
		err = gerror.Newf("msg id[%d] out of range", id)
		return
	}
	n = rt.handlers[id]
	if nil == n {
		err = gerror.Newf("msg handler[%d] not found", id)
		return
	}
	return
}
