package router

import (
	"server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/flag"
	"server/pkg/gerror"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/gmetrics"
	"server/pkg/gnet/trace"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type MsgHandler struct {
	pool       sync.Pool
	createFunc func() pb.VTMessage
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

// Register 注册消息
func (rt *MsgRouter) Register(msgID uint32, cf func() pb.VTMessage, usePool bool, df func(c gctx.Context, msg proto.Message)) error {
	if flag.IsReady() {
		return gerror.New("must register before action")
	}
	if msgID >= uint32(len(rt.handlers)) {
		return gerror.New("msg id out of range")
	}

	if rt.handlers[msgID] != nil {
		return gerror.New("msg id already register")
	}

	node := &MsgHandler{
		createFunc: cf,
		HandleFunc: df,
	}

	if usePool {
		node.pool = sync.Pool{
			New: func() any {
				return cf()
			},
		}
	}
	rt.handlers[msgID] = node

	return nil
}

// HandleMsg 处理消息
func (rt *MsgRouter) Handle(c gctx.Context) error {
	node, err := rt.getHandler(c.MsgID)
	if err != nil {
		return err
	}

	msgObj := node.pool.Get()
	msgPB := msgObj.(pb.VTMessage)
	proto.Reset(msgPB)

	if len(c.Data) > 0 {
		err = msgPB.UnmarshalVT(c.Data)
		if err != nil {
			return err
		}
	}

	if trace.Rule.ShouldLog(c.MsgID, c.ActorID, c.SesID) {
		str, _ := sonic.MarshalString(msgPB)
		zap.L().Info("<<< msg.recv:",
			zap.String("msgName", msgid.MsgIDS2S_name[int32(c.MsgID)]),
			zap.String("data", str),
			zap.Inline(&c),
		)
	}
	begin := time.Now() // todo 优化

	node.HandleFunc(c, msgPB)

	node.pool.Put(msgObj)

	cost := time.Since(begin)
	if cost > 10*time.Millisecond {
		gmetrics.GetHandlerLatencyMetric(c.MsgID).Update(float64(cost.Milliseconds()))
	}

	return nil
}

func (rt *MsgRouter) getHandler(id uint32) (n *MsgHandler, err error) {
	if id >= uint32(len(rt.handlers)) {
		err = gerror.New("msg id out of range")
		return
	}
	n = rt.handlers[id]
	if nil == n {
		err = gerror.New("msg handler not found")
		return
	}
	return
}
