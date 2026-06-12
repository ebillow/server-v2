package router

import (
	"fmt"
	"server/api/pb"
	"server/pkg/gerror"
	"server/pkg/gnet/gmetrics"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/pkg"
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

// --------------------------------------------------------------------

type IHandler interface {
	Handle(pkg.Packet) error
}
type MsgHandler struct {
	factory    MsgFactory
	HandleFunc func(pkg.Packet, pb.VTMessage)
}

// Handle 处理消息
func (h *MsgHandler) Handle(p pkg.Packet) error {
	msgPB := h.factory.Get()

	if len(p.Data) > 0 {
		err := msgPB.UnmarshalVT(p.Data)
		if err != nil {
			h.factory.Put(msgPB)
			return err
		}
	}

	p.Data = nil

	if trace.Rule.ShouldLog(p.Head.MsgID, p.Head.ActorID, p.Head.SesID) {
		str, _ := sonic.MarshalString(msgPB)
		zap.L().Info("recv",
			zap.String("type", fmt.Sprintf("%T", msgPB)),
			zap.String("data", str),
			zap.Inline(&p),
		)
	}
	begin := time.Now() // todo 优化

	h.HandleFunc(p, msgPB)

	h.factory.Put(msgPB)

	cost := time.Since(begin)
	if cost > 10*time.Millisecond {
		gmetrics.GetHandlerLatencyMetric(p.Head.MsgID).Update(float64(cost.Milliseconds()))
	}

	return nil
}

type RpcHandler struct {
	reqCreate  MsgFactory
	resCreate  MsgFactory
	HandleFunc func(pkg.Packet, pb.VTMessage, pb.VTMessage)
}

// Handle 处理消息
func (h *RpcHandler) Handle(p pkg.Packet) error {
	req := h.reqCreate.Get()

	if len(p.Data) > 0 {
		err := req.UnmarshalVT(p.Data)
		if err != nil {
			h.reqCreate.Put(req)
			return err
		}
	}

	p.Data = nil

	shouldLog := trace.Rule.ShouldLog(p.Head.MsgID, p.Head.ActorID, p.Head.SesID)
	if shouldLog {
		str, _ := sonic.MarshalString(req)
		zap.L().Info("recv rpc",
			zap.String("type", fmt.Sprintf("%T", req)),
			zap.String("data", str),
			zap.Inline(&p),
		)
	}

	begin := time.Now() // todo 优化

	res := h.resCreate.Get()
	h.HandleFunc(p, req, res)

	err := msgq.RpcRespond(&msgq.Q, p.Reply(), res)
	if err != nil {
		h.reqCreate.Put(req)
		h.resCreate.Put(res)
		return gerror.Wrapf(err, "respond %d", p.Head.MsgID)
	}
	if shouldLog {
		str, _ := sonic.MarshalString(res)
		zap.L().Info("respond rpc",
			zap.String("type", fmt.Sprintf("%T", res)),
			zap.String("data", str),
			zap.Inline(&p),
		)
	}

	h.reqCreate.Put(req)
	h.resCreate.Put(res)

	cost := time.Since(begin)
	if cost > 10*time.Millisecond {
		gmetrics.GetHandlerLatencyMetric(p.Head.MsgID).Update(float64(cost.Milliseconds()))
	}

	return nil
}
