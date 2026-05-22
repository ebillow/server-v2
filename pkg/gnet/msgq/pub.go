package msgq

import (
	"encoding/binary"
	"server/api/pb"
	"server/pkg/gnet/gctx"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Send 指定发送
func (bs *DataBus) Send(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getIdxPubBatcher(serType, serID).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSerID: bs.serID,
		FromSer:   bs.serType,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
	})
}

func (bs *DataBus) ForwardToRole(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getIdxPubBatcher(serType, serID).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSerID: bs.serID,
		FromSer:   bs.serType,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
		Flag:      gctx.Forward,
	})
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getGroupPubBatcher(serType).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ActorID:   actorID,
		SesID:     sesID,
	})
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getAllPubBatcher(serType).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ActorID:   actorID,
		SesID:     sesID,
	})
}

func (bs *DataBus) Relay(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) {
	bs.getGroupPubBatcher(pb.Server_Center).Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
		Flag:      gctx.Forward,
	})
}

func (bs *DataBus) getIdxPubBatcher(serType pb.Server, serID uint8) *PubBatcher {
	key := (uint32(serType) << 16) | uint32(serID)
	val, ok := bs.pubIdx.Load(key)
	if !ok {
		subject := getIndexSubject(serType, serID)
		tb := NewPubBatcher(subject, bs.conn)
		actual, _ := bs.pubIdx.LoadOrStore(key, tb)
		val = actual
	}
	return val.(*PubBatcher)
}

func (bs *DataBus) getGroupPubBatcher(serType pb.Server) *PubBatcher {
	key := uint32(serType)
	val, ok := bs.pubGroup.Load(key)
	if !ok {
		subject := getGroupSubject(serType)
		tb := NewPubBatcher(subject, bs.conn)
		actual, _ := bs.pubGroup.LoadOrStore(key, tb)
		val = actual
	}
	return val.(*PubBatcher)
}

func (bs *DataBus) getAllPubBatcher(serType pb.Server) *PubBatcher {
	key := uint32(serType)
	val, ok := bs.pubAll.Load(key)
	if !ok {
		subject := getAllSubject(serType)
		tb := NewPubBatcher(subject, bs.conn)
		actual, _ := bs.pubAll.LoadOrStore(key, tb)
		val = actual
	}
	return val.(*PubBatcher)
}

// PubBatcher 针对单个目标服务器的流式批处理器
type PubBatcher struct {
	subject string
	conn    *nats.Conn

	mtx   sync.Mutex
	buf   *[]byte // 从 bufPool 获取的共享大缓冲
	count int     // 当前缓冲了多少条消息
}

func NewPubBatcher(subject string, conn *nats.Conn) *PubBatcher {
	tb := &PubBatcher{
		subject: subject,
		conn:    conn,
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
	*tb.buf = binary.LittleEndian.AppendUint64(*tb.buf, ctx.ActorID)
	*tb.buf = binary.LittleEndian.AppendUint64(*tb.buf, ctx.SesID)
	*tb.buf = append(*tb.buf, ctx.FromSer)
	*tb.buf = append(*tb.buf, ctx.FromSerID)
	*tb.buf = append(*tb.buf, ctx.ToSer)
	*tb.buf = append(*tb.buf, ctx.ToSerID)
	*tb.buf = append(*tb.buf, ctx.Flag)
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

func (tb *PubBatcher) flush(data []byte, p *[]byte) {
	if p != nil {
		err := tb.conn.Publish(tb.subject, data)
		if err != nil {
			zap.L().Error("publish err", zap.Error(err))
		}
		FreeBuffer(p)
	}
}
