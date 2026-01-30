package router

import (
	"google.golang.org/protobuf/proto"
)

type msgHandler struct {
	createFunc func() proto.Message
	handleFunc func(proto.Message, Context)
}

// MsgRouter 消息处理器
type MsgRouter struct {
	handlers []*msgHandler
}

// NewMsgRouter createRoute
func NewMsgRouter(size uint32) *MsgRouter {
	r := &MsgRouter{
		make([]*msgHandler, size),
	}
	return r
}

// Register 注册消息
func (rt *MsgRouter) Register(msgID uint32, cf func() proto.Message, df func(msg proto.Message, c Context)) error {
	if int(msgID) >= len(rt.handlers) {
		return errMsgIDBiggerThanMax
	}
	rt.handlers[msgID] = &msgHandler{
		createFunc: cf,
		handleFunc: df,
	}

	return nil
}

// Handle 处理消息
func (rt *MsgRouter) HandleMsg(msgID uint32, msgData []byte, c Context) (proto.Message, error) {
	node, err := rt.getHandler(msgID)
	if err != nil {
		return nil, err
	}

	msgPb, err := rt.parseMsg(node, msgData)
	if err != nil {
		return nil, err
	}

	node.handleFunc(msgPb, c)
	return msgPb, nil
}

func (rt *MsgRouter) getHandler(id uint32) (n *msgHandler, err error) {
	if int(id) >= len(rt.handlers) {
		err = errAPINotFind
		return
	}

	n = rt.handlers[id]
	if nil == n || nil == n.createFunc {
		err = errAPINotFind
		return
	}
	return
}

func (rt *MsgRouter) parseMsg(n *msgHandler, data []byte) (msg proto.Message, err error) {
	msg = n.createFunc()
	if msg == nil { // 允许只有消息id没内容
		return
	}
	err = proto.Unmarshal(data, msg)
	return
}
