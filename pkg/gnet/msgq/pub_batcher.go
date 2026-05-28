package msgq

import (
	"encoding/binary"
	"server/api/pb"
	"server/pkg/gnet/gctx"
	"server/pkg/queue"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func (bs *DataBus) getIdxPubBatcher(serType pb.Server, serID uint8) (*PubBatcher, error) {
	if serType >= SvcTypeMax || serID >= SvcIDMax {
		return nil, ErrArg
	}
	tb := bs.pubIDXs[serType][serID].Load()
	if tb != nil {
		return tb, nil
	}

	bs.pubIDXMtx.Lock()
	defer bs.pubIDXMtx.Unlock()

	if tb = bs.pubIDXs[serType][serID].Load(); tb != nil {
		return tb, nil
	}

	subject := idxSubjectName(serType, serID)
	tb = NewPubBatcher(subject, bs.conn)
	bs.pubIDXs[serType][serID].Store(tb)

	return tb, nil
}

func (bs *DataBus) getGroupPubBatcher(serType pb.Server) (*PubBatcher, error) {
	if serType >= SvcTypeMax {
		return nil, ErrArg
	}
	tb := bs.pubGroup[serType].Load()
	if tb != nil {
		return tb, nil
	}

	bs.pubGroupMtx.Lock()
	defer bs.pubGroupMtx.Unlock()

	if tb = bs.pubGroup[serType].Load(); tb != nil {
		return tb, nil
	}

	subject := groupSubjectName(serType)
	tb = NewPubBatcher(subject, bs.conn)
	bs.pubGroup[serType].Store(tb)

	return tb, nil
}

func (bs *DataBus) getAllPubBatcher(serType pb.Server) (*PubBatcher, error) {
	if serType >= SvcTypeMax {
		return nil, ErrArg
	}

	tb := bs.pubAll[serType].Load()
	if tb != nil {
		return tb, nil
	}

	bs.pubAllMtx.Lock()
	defer bs.pubAllMtx.Unlock()

	if tb = bs.pubAll[serType].Load(); tb != nil {
		return tb, nil
	}

	subject := allSubjectName(serType)
	tb = NewPubBatcher(subject, bs.conn)
	bs.pubAll[serType].Store(tb)

	return tb, nil
}

func (bs *DataBus) flushAllBatchers() {
	for i := range bs.pubIDXs {
		for j := range bs.pubIDXs[i] {
			if tb := bs.pubIDXs[i][j].Load(); tb != nil {
				tb.StopAndFlush()
			}
		}
	}

	for i := range bs.pubGroup {
		if tb := bs.pubGroup[i].Load(); tb != nil {
			tb.StopAndFlush()
		}
	}

	for i := range bs.pubAll {
		if tb := bs.pubAll[i].Load(); tb != nil {
			tb.StopAndFlush()
		}
	}
}

// PubBatcher 针对单个目标服务器的流式批处理器
type PubBatcher struct {
	subject string
	conn    *nats.Conn

	queue *queue.SwapQueue[gctx.Context]

	closed atomic.Bool
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewPubBatcher(subject string, conn *nats.Conn) *PubBatcher {
	tb := &PubBatcher{
		subject: subject,
		conn:    conn,
		stopCh:  make(chan struct{}),
		queue:   queue.NewSwapQueue[gctx.Context](4096, 40960),
	}

	tb.startLoop()

	return tb
}

func (tb *PubBatcher) Add(ctx gctx.Context) error {
	return tb.queue.Push(ctx)
}

// 定时器：每 25 毫秒强制发送一次，防止消息一直积压不发
func (tb *PubBatcher) startLoop() {
	const maxRetainBufSize = buffSize * 2
	tb.wg.Add(1)
	go func() {
		ticker := time.NewTicker(time.Millisecond * 25)
		defer func() {
			ticker.Stop()
			defer tb.wg.Done()
		}()

		buf := make([]byte, 0, buffSize)
		count := 0

		flush := func() {
			if count > 0 {
				err := tb.conn.Publish(tb.subject, buf)
				if err != nil {
					zap.L().Error("nats publish err", zap.Error(err))
				}
				if cap(buf) > maxRetainBufSize {
					buf = make([]byte, 0, buffSize)
				} else {
					buf = buf[:0]
				}
				count = 0
			}
		}

		processFunc := func(ctx gctx.Context) {
			subSize := headerSize + len(ctx.Data)

			// 追加前缀
			buf = binary.LittleEndian.AppendUint32(buf, uint32(subSize))

			// 追加主体
			buf = binary.LittleEndian.AppendUint32(buf, ctx.MsgID)
			buf = binary.LittleEndian.AppendUint64(buf, ctx.ActorID)
			buf = binary.LittleEndian.AppendUint64(buf, ctx.SesID)
			buf = append(buf, ctx.FromSer, ctx.FromSerID, ctx.ToSer, ctx.ToSerID, ctx.Flag)
			buf = append(buf, ctx.Data...)

			count++

			// 如果在 Range 遍历期间达到了批处理上限，直接触发发送
			if count >= batchCount || len(buf) > buffSize {
				flush()
			}
		}

		for {
			select {
			case <-tb.queue.Sig():
				tb.queue.Range(processFunc)
			case <-ticker.C:
				tb.queue.Range(processFunc)
				flush()
			case <-tb.stopCh:
				tb.queue.Range(processFunc)
				flush()
				return
			}
		}
	}()
}

// StopAndFlush 关闭并清空，关闭前通过了closed判断但还没写入queue的，可能没发出去。要解决要加RWMutex，有需求再加
func (tb *PubBatcher) StopAndFlush() {
	if tb.closed.CompareAndSwap(false, true) {
		close(tb.stopCh) // 通知后台协程退出
		tb.wg.Wait()     // 阻塞等待数据发送完毕
	}
}
