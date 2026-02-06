package router

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/trace"
	"server/pkg/logger"
)

type MsgHandler struct {
	createFunc func() proto.Message
	HandleFunc func(proto.Message, gctx.Context)
}

// MsgRouter 消息处理器
type MsgRouter struct {
	handlers map[uint32]*MsgHandler
}

// NewMsgRouter createRoute
func NewMsgRouter() *MsgRouter {
	r := &MsgRouter{
		make(map[uint32]*MsgHandler),
	}
	return r
}

// Register 注册消息
func (rt *MsgRouter) Register(msgID uint32, cf func() proto.Message, df func(msg proto.Message, c gctx.Context)) error {
	rt.handlers[msgID] = &MsgHandler{
		createFunc: cf,
		HandleFunc: df,
	}

	return nil
}

// HandleMsg 处理消息
func (rt *MsgRouter) HandleMsg(c gctx.Context) error {
	node, err := rt.GetHandler(c.Msg.MsgID)
	if err != nil {
		return err
	}

	msgPB, err := rt.ParseMsg(node, c.Msg.Data)
	if err != nil {
		return err
	}

	if trace.Rule.ShouldLog(c.Msg.MsgID, c.Msg.RoleID, c.Msg.SesID) {
		info := "<<< msg.recv:"
		zap.L().Info(info,
			zap.Inline(c),
			zap.Any("data", msgPB),
			logger.Blue.Field(),
		)
	}

	node.HandleFunc(msgPB, c)
	return nil
}

func (rt *MsgRouter) GetHandler(id uint32) (n *MsgHandler, err error) {
	n = rt.handlers[id]
	if nil == n || nil == n.createFunc {
		err = errAPINotFind
		return
	}
	return
}

func (rt *MsgRouter) ParseMsg(n *MsgHandler, data []byte) (msg proto.Message, err error) {
	msg = n.createFunc()
	if msg == nil { // 允许只有消息id没内容
		return
	}
	err = proto.Unmarshal(data, msg)
	return
}
