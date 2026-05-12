package router

import (
	"server/pkg/gerror"
	"server/pkg/gnet/gctx"
	"time"

	"google.golang.org/protobuf/proto"
)

type MsgHandler struct {
	createFunc func() proto.Message
	HandleFunc func(gctx.Context, proto.Message)
}

// MsgRouter 消息处理器
type MsgRouter struct {
	handlers []*MsgHandler
}

// NewMsgRouter createRoute
func NewMsgRouter(max int32) *MsgRouter {
	r := &MsgRouter{
		make([]*MsgHandler, max),
	}
	return r
}

// register 注册消息
func (rt *MsgRouter) register(msgID uint32, cf func() proto.Message, df func(c gctx.Context, msg proto.Message)) error {
	if msgID >= uint32(len(rt.handlers)) {
		return gerror.New("msg id out of range")
	}

	if rt.handlers[msgID] != nil {
		return gerror.New("msg id already register")
	}

	rt.handlers[msgID] = &MsgHandler{
		createFunc: cf,
		HandleFunc: df,
	}

	return nil
}

// handleMsg 处理消息
func (rt *MsgRouter) handleMsg(c gctx.Context, logFunc func(message proto.Message)) error {
	node, err := rt.getHandler(c.MsgID)
	if err != nil {
		return err
	}

	msgPB, err := rt.parseMsg(node, c.Data)
	if err != nil {
		return err
	}

	logFunc(msgPB)
	begin := time.Now()

	node.HandleFunc(c, msgPB)

	span := time.Since(begin)
	if span.Milliseconds() > 200 {
		// zap.L().Warn("handle msg timeout:",
		// 	zap.Inline(c),
		// 	zap.Duration("cost", span),
		// )
	}

	return nil
}

func (rt *MsgRouter) getHandler(id uint32) (n *MsgHandler, err error) {
	if id >= uint32(len(rt.handlers)) {
		err = gerror.New("msg id out of range")
		return
	}
	n = rt.handlers[id]
	if nil == n || nil == n.createFunc {
		err = gerror.New("msg handler not found")
		return
	}
	return
}

func (rt *MsgRouter) parseMsg(n *MsgHandler, data []byte) (msg proto.Message, err error) {
	msg = n.createFunc()
	if msg == nil { // 允许只有消息id没内容
		return
	}
	err = proto.Unmarshal(data, msg)
	return
}
