package js

import (
	"context"
	"server/api/pb"
	"server/pkg/gnet/batcher"
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gctx"
	"server/pkg/thread"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

func (jt *JetStream) Send(serType pb.Server, serID uint8, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	if jt.closed.Load() {
		return dep.ErrClosed
	}
	pbt, err := jt.getIdxPubBatcher(serType, serID)
	if err != nil {
		return err
	}

	return pbt.Add(gctx.Context{
		Data:      data,
		ActorID:   roleID,
		SesID:     sesID,
		MsgID:     msgID,
		FromSerID: jt.serID,
		FromSer:   jt.serType,
		ToSerID:   serID,
		ToSer:     uint8(serType),
	})
}

func (jt *JetStream) SendAny(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	if jt.closed.Load() {
		return dep.ErrClosed
	}
	pbt, err := jt.getGroupPubBatcher(serType)
	if err != nil {
		return err
	}

	return pbt.Add(gctx.Context{
		Data:      data,
		ActorID:   roleID,
		SesID:     sesID,
		MsgID:     msgID,
		FromSerID: jt.serID,
		FromSer:   jt.serType,
	})
}

func (jt *JetStream) getIdxPubBatcher(serType pb.Server, serID uint8) (*PubBatcher, error) {
	if serType >= SvcTypeMax || serID >= SvcIDMax {
		return nil, dep.ErrArg
	}
	tb := jt.pubIDXs[serType][serID].Load()
	if tb != nil {
		return tb, nil
	}

	jt.pubIDXMtx.Lock()
	defer jt.pubIDXMtx.Unlock()

	if tb = jt.pubIDXs[serType][serID].Load(); tb != nil {
		return tb, nil
	}

	subject := getIndexSubject(serType, serID)
	tb = NewPubBatcher(jt.ctx, subject, jt.JS)
	jt.pubIDXs[serType][serID].Store(tb)

	return tb, nil
}

func (jt *JetStream) getGroupPubBatcher(serType pb.Server) (*PubBatcher, error) {
	if serType >= SvcTypeMax {
		return nil, dep.ErrArg
	}
	tb := jt.pubGroup[serType].Load()
	if tb != nil {
		return tb, nil
	}

	jt.pubGroupMtx.Lock()
	defer jt.pubGroupMtx.Unlock()

	if tb = jt.pubGroup[serType].Load(); tb != nil {
		return tb, nil
	}

	subject := getGroupSubject(serType)
	tb = NewPubBatcher(jt.ctx, subject, jt.JS)
	jt.pubGroup[serType].Store(tb)

	return tb, nil
}

func (jt *JetStream) flushAllBatchers() {
	for i := range jt.pubIDXs {
		for j := range jt.pubIDXs[i] {
			if tb := jt.pubIDXs[i][j].Load(); tb != nil {
				tb.StopAndFlush()
			}
		}
	}

	for i := range jt.pubGroup {
		if tb := jt.pubGroup[i].Load(); tb != nil {
			tb.StopAndFlush()
		}
	}
}

type Ack struct {
	Future jetstream.PubAckFuture
	PBuf   *[]byte
}

// PubBatcher 针对单个目标服务器的流式批处理器
type PubBatcher struct {
	*batcher.BaseBatcher
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

	tb.BaseBatcher = batcher.NewBaseBatcher(func(data []byte, p *[]byte, count int) {
		if p == nil {
			return
		}
		ackF, err := tb.JS.PublishMsgAsync(&nats.Msg{
			Subject: tb.subject,
			Data:    data,
		})
		if err != nil {
			zap.L().Error("js publish msg", zap.Error(err))
			batcher.FreeBuffer(p)
			return
		}
		select {
		case tb.ack <- Ack{Future: ackF, PBuf: p}:
		default:
			zap.L().Warn("js publish msg chan full")
			batcher.FreeBuffer(p)
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
	defer func() {
		if task.PBuf != nil {
			batcher.FreeBuffer(task.PBuf)
		}
	}()

	select {
	case <-task.Future.Ok():
	case err := <-task.Future.Err():
		zap.L().Error("NATS async pub error", zap.Error(err), zap.String("subject", tb.subject))
	case <-time.After(3 * time.Second):
		zap.L().Error("NATS async pub timeout", zap.String("subject", tb.subject))
	}
}
