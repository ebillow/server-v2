package snet

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/internal/pb/msgid"
	"server/internal/util"
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
func newRoleMsgRouter(size int) *roleMsgRouter {
	r := &roleMsgRouter{
		make([]*msgHandler, size),
	}
	return r
}

// register 注册消息
func (rt *roleMsgRouter) register(msgID uint16, cf func() proto.Message, df func(msg proto.Message, r *role.Role)) {
	rt.handlers[msgID] = &msgHandler{
		createFunc: cf,
		handleFunc: df,
	}
}

// Handle 处理消息
func (rt *roleMsgRouter) Handle(msg *nats.Msg, r *role.Role) error {
	msgID := util.ParseUint32(msg.Header.Get("msg_id"))
	node, err := rt.getHandler(msgID)
	if err != nil {
		zap.L().Warn("can not find msg  roleMsgRouter", zap.Uint32("msg_id", msgID), zap.Error(err), zap.Inline(r))
		return err
	}

	msgPb, err := rt.parseMsg(node, msg.Data)
	if err != nil {
		zap.L().Warn("parse msg error", zap.Uint32("msg_id", msgID), zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]), zap.Error(err), zap.Inline(r))
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
