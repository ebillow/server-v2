package snet

import (
	"google.golang.org/protobuf/proto"
	"server/game/role"
)

type msgHandler struct {
	createFunc func() proto.Message
	handleFunc func(msg proto.Message, r *role.Role)
}

// roleMsgRouter 消息处理器
type roleMsgRouter struct {
	handlers []*msgHandler
}

// newRoleMsgRouter createRoute
func newRoleMsgRouter(size uint32) *roleMsgRouter {
	r := &roleMsgRouter{
		make([]*msgHandler, size),
	}
	return r
}

// Register 注册消息
func (rt *roleMsgRouter) register(msgID uint32, cf func() proto.Message, df func(msg proto.Message, r *role.Role)) error {
	if int(msgID) >= len(rt.handlers) {
		return errMsgIDBigThanMax
	}
	rt.handlers[msgID] = &msgHandler{
		createFunc: cf,
		handleFunc: df,
	}

	return nil
}

// Handle 处理消息
func (rt *roleMsgRouter) handle(msgID uint32, msgData []byte, r *role.Role) error {
	node, err := rt.getHandler(msgID)
	if err != nil {
		return err
	}

	msgPb, err := rt.parseMsg(node, msgData)
	if err != nil {
		return err
	}

	node.handleFunc(msgPb, r)

	return nil
}

func (rt *roleMsgRouter) getHandler(id uint32) (n *msgHandler, err error) {
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

func (rt *roleMsgRouter) parseMsg(n *msgHandler, data []byte) (msg proto.Message, err error) {
	msg = n.createFunc()
	if msg == nil { // 允许只有消息id没内容
		return
	}
	err = proto.Unmarshal(data, msg)
	return
}
