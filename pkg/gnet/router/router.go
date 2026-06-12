package router

import (
	"server/api/pb"
	"server/pkg/flag"
	"server/pkg/gerror"
	"server/pkg/gnet/gmsg"
	"sync"
)

// Router 消息处理器
type Router struct {
	handlers []IHandler
}

// NewMsgRouter createRoute
func NewMsgRouter(max int32) *Router {
	r := &Router{
		make([]IHandler, max),
	}
	return r
}

// Register 注册消息
func (rt *Router) Register(msgID uint32, cf func() pb.VTMessage, usePool bool, df func(gmsg.Message, pb.VTMessage)) error {
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

func (rt *Router) RegisterRpc(msgID uint32, createReq func() pb.VTMessage, createRes func() pb.VTMessage, usePool bool, df func(p gmsg.Message, req pb.VTMessage, res pb.VTMessage)) error {
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

func (rt *Router) Handle(p gmsg.Message) error {
	handler, err := rt.getHandler(p.Head.MsgID)
	if err != nil {
		return err
	}
	return handler.Handle(p)
}

func (rt *Router) getHandler(id uint32) (n IHandler, err error) {
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
