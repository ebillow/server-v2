package jetq

import (
	"context"
	"server/api/pb"
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/pub"
	"server/pkg/thread"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

func (jt *JetStream) SendToNode(serType pb.Server, serID uint8, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	if jt.closed.Load() {
		return dep.ErrClosed
	}
	pbt, err := jt.pub.Node(serType, serID)
	if err != nil {
		return err
	}

	return pbt.Add(gmsg.Message{
		Head: gmsg.Head{
			ActorID:   roleID,
			SesID:     sesID,
			MsgID:     msgID,
			FromSerID: jt.serID,
			FromSer:   jt.serType,
			ToSerID:   serID,
			ToSer:     uint8(serType),
		},
		Data: data,
	})
}

func (jt *JetStream) SendToAny(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	if jt.closed.Load() {
		return dep.ErrClosed
	}
	pbt, err := jt.pub.Any(serType)
	if err != nil {
		return err
	}

	return pbt.Add(gmsg.Message{
		Head: gmsg.Head{
			ActorID:   roleID,
			SesID:     sesID,
			MsgID:     msgID,
			FromSerID: jt.serID,
			FromSer:   jt.serType,
		},
		Data: data,
	})
}

type Ack struct {
	Future  jetstream.PubAckFuture
	Subject string
}

// PubBatcher 针对单个目标服务器的流式批处理器
type PubBatcher struct {
	*pub.Batcher
	subject string
	JS      jetstream.JetStream
	ack     chan Ack
}

func NewPubBatcher(ctx context.Context, subject string, js jetstream.JetStream) *PubBatcher {
	tb := &PubBatcher{
		subject: subject,
		JS:      js,
		ack:     make(chan Ack, 40960),
	}

	tb.Batcher = pub.NewBatcher(func(data []byte, count int) {
		if len(data) == 0 {
			return
		}

		ackF, err := tb.JS.PublishMsgAsync(&nats.Msg{
			Subject: tb.subject,
			Data:    data,
		})
		if err != nil {
			zap.L().Error("jetq publish msg", zap.Error(err))
			return
		}
		select {
		case tb.ack <- Ack{Future: ackF, Subject: subject}:
		default:
			zap.L().Warn("jetq publish msg chan full")
		}
	})

	for i := 0; i < 2; i++ {
		thread.GoSafe(func() {
			tb.ackWorker(ctx)
		})
	}

	return tb
}

func (tb *PubBatcher) ackWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return // 退出协程
		case task := <-tb.ack:
			tb.waitAck(task)
		}
	}
}

func (tb *PubBatcher) waitAck(task Ack) {
	timer := time.NewTimer(time.Second * 3)
	defer timer.Stop()

	select {
	case <-task.Future.Ok():
	case err := <-task.Future.Err():
		zap.L().Error("NATS async pub error", zap.Error(err), zap.String("subject", tb.subject))
	case <-timer.C:
		zap.L().Error("NATS async pub timeout", zap.String("subject", tb.subject))
	}
}

func (jt *JetStream) NewNodePub(serType pb.Server, serID uint8) *PubBatcher {
	return NewPubBatcher(jt.ctx, getNodeSubject(serType, serID), jt.JS)
}

func (jt *JetStream) NewAnyPub(serType pb.Server) *PubBatcher {
	return NewPubBatcher(jt.ctx, getAnySubject(serType), jt.JS)
}

func (jt *JetStream) NewBroadcastPub(serType pb.Server) *PubBatcher {
	panic("jetq can not use broadcast pub batcher")
	return nil
}
