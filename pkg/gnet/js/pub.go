package js

import (
	"encoding/binary"
	"server/pkg/gnet/gctx"
	"server/pkg/pb"
	"server/pkg/thread"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

func (jt *JetStream) Send(serType pb.Server, serID uint8, msgID uint32, data []byte, roleID uint64, sesID uint64) {
	jt.getIdxPubBatcher(serType, serID).Add(gctx.Context{
		Data:      data,
		RoleID:    roleID,
		SesID:     sesID,
		MsgID:     msgID,
		FromSerID: jt.serID,
		FromSer:   jt.serType,
		ToSerID:   serID,
		ToSer:     uint8(serType),
	})
}

func (jt *JetStream) SendAny(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) {
	jt.getGroupPubBatcher(serType).Add(gctx.Context{
		Data:      data,
		RoleID:    roleID,
		SesID:     sesID,
		MsgID:     msgID,
		FromSerID: jt.serID,
		FromSer:   jt.serType,
	})
}

func (jt *JetStream) getIdxPubBatcher(serType pb.Server, serID uint8) *PubBatcher {
	key := (uint32(serType) << 16) | uint32(serID)
	val, ok := jt.pubIdx.Load(key)
	if !ok {
		tb := NewPubBatcher(getIndexSubject(serType, serID), jt.JS)
		actual, loaded := jt.pubIdx.LoadOrStore(key, tb)
		if loaded {
			tb.Stop()
		}
		val = actual
	}
	return val.(*PubBatcher)
}

func (jt *JetStream) getGroupPubBatcher(serType pb.Server) *PubBatcher {
	key := uint64(serType)
	val, ok := jt.pubGroup.Load(key)
	if !ok {
		subject := getGroupSubject(serType)
		tb := NewPubBatcher(subject, jt.JS)
		actual, _ := jt.pubGroup.LoadOrStore(key, tb)
		val = actual
	}
	return val.(*PubBatcher)
}

// PubBatcher 针对单个目标服务器的流式批处理器
type PubBatcher struct {
	subject string
	JS      jetstream.JetStream

	mtx    sync.Mutex
	buf    *[]byte // 从 bufPool 获取的共享大缓冲
	count  int     // 当前缓冲了多少条消息
	ack    chan Ack
	closed chan struct{}
}

func NewPubBatcher(subject string, js jetstream.JetStream) *PubBatcher {
	tb := &PubBatcher{
		subject: subject,
		JS:      js,
		ack:     make(chan Ack, 40960),
		closed:  make(chan struct{}),
	}

	for i := 0; i < 10; i++ {
		thread.GoSafe(func() {
			tb.ackWorker()
		})
	}

	tb.startTicker()
	return tb
}

// Add 立即序列化并追加到批处理缓冲中
func (tb *PubBatcher) Add(ctx gctx.Context) {
	var flushData []byte
	var flushBp *[]byte

	tb.mtx.Lock()
	if tb.buf == nil {
		tb.buf = GetBuffer()
		*tb.buf = (*tb.buf)[:0]
	}

	subSize := headerSize + len(ctx.Data)

	// 写入 4 字节的长度前缀
	*tb.buf = binary.LittleEndian.AppendUint32(*tb.buf, uint32(subSize))

	// 写入消息本体
	*tb.buf = binary.LittleEndian.AppendUint32(*tb.buf, ctx.MsgID)
	*tb.buf = binary.LittleEndian.AppendUint64(*tb.buf, ctx.RoleID)
	*tb.buf = binary.LittleEndian.AppendUint64(*tb.buf, ctx.SesID)
	*tb.buf = append(*tb.buf, ctx.FromSer)
	*tb.buf = append(*tb.buf, ctx.FromSerID)
	*tb.buf = append(*tb.buf, ctx.ToSer)
	*tb.buf = append(*tb.buf, ctx.ToSerID)
	*tb.buf = append(*tb.buf, ctx.Forward)
	*tb.buf = append(*tb.buf, ctx.Data...)

	tb.count++

	if tb.count >= batchCount || len(*tb.buf) > buffSize {
		flushData = *tb.buf
		flushBp = tb.buf
		tb.buf = nil // 置空，下一个包进来会重新从 pool 取
		tb.count = 0
	}
	tb.mtx.Unlock()

	tb.flush(flushData, flushBp)
}

type Ack struct {
	Future jetstream.PubAckFuture
	PBuf   *[]byte
}

// 定时器：每 25 毫秒强制发送一次，防止消息一直积压不发
func (tb *PubBatcher) startTicker() {
	go func() {
		ticker := time.NewTicker(time.Millisecond * 25)
		defer ticker.Stop()

		for range ticker.C {
			var flushData []byte
			var flushBp *[]byte

			tb.mtx.Lock()
			if tb.count > 0 && tb.buf != nil {
				flushData = *tb.buf
				flushBp = tb.buf
				tb.buf = nil
				tb.count = 0
			}
			tb.mtx.Unlock()

			tb.flush(flushData, flushBp)
		}
	}()
}

func (tb *PubBatcher) Stop() {
	close(tb.closed)
}

func (tb *PubBatcher) ackWorker() {
	for {
		select {
		case <-tb.closed:
			return // 退出协程
		case task := <-tb.ack:
			select {
			case <-task.Future.Ok():
			case err := <-task.Future.Err():
				zap.L().Error("NATS async pub error", zap.Error(err))
			case <-time.After(3 * time.Second):
				zap.L().Error("NATS async pub timeout")
			}

			if task.PBuf != nil {
				FreeBuffer(task.PBuf)
			}
		}
	}
}

func (tb *PubBatcher) flush(data []byte, p *[]byte) {
	if p != nil {
		ackF, err := tb.JS.PublishMsgAsync(&nats.Msg{
			Subject: tb.subject,
			Data:    data,
		})
		if err != nil {
			zap.L().Error("js publish msg", zap.Error(err))
			return
		}
		select {
		case tb.ack <- Ack{Future: ackF, PBuf: p}:
		default:
			zap.L().Warn("js publish msg chan full")
			FreeBuffer(p)
		}
	}
}
