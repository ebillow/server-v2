package router

import (
	"fmt"
	"server/api/pb"
	"server/pkg/gerror"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/gmetrics"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type MsgFactory interface {
	Get() pb.VTMessage
	Put(pb.VTMessage)
}

// 池化工厂
type poolFactory struct {
	pool sync.Pool
}

func (f *poolFactory) Get() pb.VTMessage {
	msg := f.pool.Get().(pb.VTMessage)
	proto.Reset(msg)
	return msg
}

func (f *poolFactory) Put(msg pb.VTMessage) {
	f.pool.Put(msg)
}

// 非池化工厂
type newFactory struct {
	create func() pb.VTMessage
}

func (f *newFactory) Get() pb.VTMessage { return f.create() }
func (f *newFactory) Put(pb.VTMessage)  {} // no-op

type IHandler interface {
	Handle(c gctx.Context) error
}
type MsgHandler struct {
	factory    MsgFactory
	HandleFunc func(gctx.Context, pb.VTMessage)
}

// Handle 处理消息
func (h *MsgHandler) Handle(c gctx.Context) error {
	msgPB := h.factory.Get()

	if len(c.Data) > 0 {
		err := msgPB.UnmarshalVT(c.Data)
		if err != nil {
			h.factory.Put(msgPB)
			return err
		}
	}

	if trace.Rule.ShouldLog(c.MsgID, c.ActorID, c.SesID) {
		str, _ := sonic.MarshalString(msgPB)
		zap.L().Info("recv",
			zap.String("type", fmt.Sprintf("%T", msgPB)),
			zap.String("data", str),
			zap.Inline(&c),
		)
	}
	begin := time.Now() // todo 优化

	h.HandleFunc(c, msgPB)

	h.factory.Put(msgPB)

	cost := time.Since(begin)
	if cost > 10*time.Millisecond {
		gmetrics.GetHandlerLatencyMetric(c.MsgID).Update(float64(cost.Milliseconds()))
	}

	return nil
}

type RpcHandler struct {
	reqCreate  MsgFactory
	resCreate  MsgFactory
	HandleFunc func(gctx.Context, pb.VTMessage, pb.VTMessage)
}

// Handle 处理消息
func (h *RpcHandler) Handle(c gctx.Context) error {
	req := h.reqCreate.Get()

	if len(c.Data) > 0 {
		err := req.UnmarshalVT(c.Data)
		if err != nil {
			h.reqCreate.Put(req)
			return err
		}
	}

	if trace.Rule.ShouldLog(c.MsgID, c.ActorID, c.SesID) {
		str, _ := sonic.MarshalString(req)
		zap.L().Info("recv",
			zap.String("type", fmt.Sprintf("%T", req)),
			zap.String("data", str),
			zap.Inline(&c),
		)
	}
	begin := time.Now() // todo 优化

	res := h.resCreate.Get()
	h.HandleFunc(c, req, res)

	err := msgq.RpcRespond(c.Raw, res)
	if err != nil {
		h.reqCreate.Put(req)
		h.resCreate.Put(res)
		return gerror.Wrapf(err, "respond %d", c.MsgID)
	}

	h.reqCreate.Put(req)
	h.resCreate.Put(res)

	cost := time.Since(begin)
	if cost > 10*time.Millisecond {
		gmetrics.GetHandlerLatencyMetric(c.MsgID).Update(float64(cost.Milliseconds()))
	}

	return nil
}
