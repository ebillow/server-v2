package msgq

import (
	"encoding/binary"
	"server/pkg/gnet/gctx"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const (
	headerSize = 4 + 8 + 8 + 4 + 4 + 1
	batchCount = 500
	buffSize   = 1024 * batchCount
)

// bufPool 序列化 Buffer 对象池
//
//	预分配
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, buffSize)
		return &b
	},
}

// FreeBuffer 发送完毕后，需要手动归还 []byte
func FreeBuffer(b *[]byte) {
	//  容量过大的异常包直接丢弃，防止占用过多常驻内存
	if cap(*b) > 10*buffSize {
		return
	}
	*b = (*b)[:0] // 重置长度为 0，但保留 capacity
	bufPool.Put(b)
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
		tb.buf = bufPool.Get().(*[]byte)
		*tb.buf = (*tb.buf)[:0]
	}

	// 1. 计算单条消息的总长度
	subSize := headerSize + len(ctx.Data)

	// 2. 【关键】写入 4 字节的长度前缀
	*tb.buf = binary.LittleEndian.AppendUint32(*tb.buf, uint32(subSize))

	// 3. 写入消息本体 (复用你原来的 Encode 逻辑)
	*tb.buf = binary.LittleEndian.AppendUint32(*tb.buf, ctx.MsgID)
	*tb.buf = binary.LittleEndian.AppendUint64(*tb.buf, ctx.RoleID)
	*tb.buf = binary.LittleEndian.AppendUint64(*tb.buf, ctx.SesID)
	*tb.buf = binary.LittleEndian.AppendUint32(*tb.buf, uint32(ctx.SerType))
	*tb.buf = binary.LittleEndian.AppendUint32(*tb.buf, uint32(ctx.SerID))
	*tb.buf = append(*tb.buf, ctx.Forward)
	*tb.buf = append(*tb.buf, ctx.Data...)

	tb.count++

	// 4. 触发策略
	if tb.count >= batchCount || len(*tb.buf) > buffSize {
		flushData = *tb.buf
		flushBp = tb.buf
		tb.buf = nil // 置空，下一个包进来会重新从 pool 取
		tb.count = 0
	}
	tb.mtx.Unlock()

	// 5. 在锁外执行 NATS 发送，避免阻塞其他 Goroutine
	tb.flush(flushData, flushBp)
}

// 定时器：每 5 毫秒强制发送一次，防止消息一直积压不发
func (tb *PubBatcher) startTicker() {
	go func() {
		ticker := time.NewTicker(time.Millisecond * 25)
		defer ticker.Stop()

		for range ticker.C {
			tb.flushAll()
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

func (tb *PubBatcher) flushAll() {
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
