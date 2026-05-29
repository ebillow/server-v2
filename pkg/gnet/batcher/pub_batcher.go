package batcher

import (
	"encoding/binary"
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gctx"
	"sync"
	"sync/atomic"
	"time"
)

type batcherState int32

const (
	BStateRunning batcherState = iota
	BStateClosing
	BStateStopped
)

// FlushFunc 是由具体的 NATS / JetStream 实现的发送回调
type FlushFunc func(data []byte, p *[]byte, count int)

// BaseBatcher 针对单个目标服务器的流式批处理器
type BaseBatcher struct {
	mtx   sync.Mutex
	buf   *[]byte // 从 bufPool 获取的共享大缓冲
	count int     // 当前缓冲了多少条消息

	// queue *queue.SwapQueue[gctx.Context]

	state atomic.Int32
	wg    sync.WaitGroup

	flushFn FlushFunc
}

func NewBaseBatcher(flushFn FlushFunc) *BaseBatcher {
	tb := &BaseBatcher{
		// queue:   queue.NewSwapQueue[gctx.Context](4096, 40960),
		flushFn: flushFn,
	}

	tb.startLoop()

	return tb
}

func (tb *BaseBatcher) Add(ctx gctx.Context) error {
	msgSize := HeaderSize + len(ctx.Data)
	frameSize := 4 + msgSize
	var tasks [2]flushTask
	taskN := 0

	tb.mtx.Lock()

	if batcherState(tb.state.Load()) != BStateRunning {
		tb.mtx.Unlock()
		return dep.ErrClosed
	}

	if tb.buf == nil {
		tb.buf = GetBuffer()
		*tb.buf = (*tb.buf)[:0]
	}

	if len(*tb.buf)+frameSize > cap(*tb.buf) && len(*tb.buf) > 0 {
		tasks[taskN] = flushTask{data: *tb.buf, bp: tb.buf, count: tb.count}
		taskN++
		tb.buf = GetBuffer()
		*tb.buf = (*tb.buf)[:0]
		tb.count = 0
	}

	if frameSize > cap(*tb.buf) {
		monsterBuf := make([]byte, frameSize)
		SerializeFrame(monsterBuf, msgSize, ctx)

		// bp 传 nil，表示不需要 FreeBuffer
		tasks[taskN] = flushTask{data: monsterBuf, bp: nil, count: 1}
		taskN++
	} else {
		buf := *tb.buf
		pos := len(buf)
		buf = buf[:pos+frameSize]

		SerializeFrame(buf[pos:pos+frameSize], msgSize, ctx)

		*tb.buf = buf
		tb.count++

		if tb.count >= batchCount || len(*tb.buf) >= buffSize {
			tasks[taskN] = flushTask{data: *tb.buf, bp: tb.buf, count: tb.count}
			taskN++
			tb.buf = nil // 置空，下一次循环会重新 GetBuffer
			tb.count = 0
		}
	}

	if taskN > 0 {
		tb.wg.Add(len(tasks))
	}
	tb.mtx.Unlock()

	for i := 0; i < taskN; i++ {
		t := tasks[i]
		tb.flushFn(t.data, t.bp, t.count)
		tb.wg.Done()
	}

	return nil
}

type flushTask struct {
	data  []byte
	bp    *[]byte // 如果是 nil，说明是临时大包，不需要回收到 pool
	count int
}

func SerializeFrame(dst []byte, msgSize int, ctx gctx.Context) {
	binary.LittleEndian.PutUint32(dst[0:], uint32(msgSize))
	binary.LittleEndian.PutUint32(dst[4:], ctx.MsgID)
	binary.LittleEndian.PutUint64(dst[8:], ctx.ActorID)
	binary.LittleEndian.PutUint64(dst[16:], ctx.SesID)
	dst[24] = ctx.FromSer
	dst[25] = ctx.FromSerID
	dst[26] = ctx.ToSer
	dst[27] = ctx.ToSerID
	dst[28] = ctx.Flag
	copy(dst[29:], ctx.Data)
}

// 定时器：每 25 毫秒强制发送一次，防止消息一直积压不发
func (tb *BaseBatcher) startLoop() {
	go func() {
		ticker := time.NewTicker(time.Millisecond * 25)
		defer func() {
			ticker.Stop()
		}()

		for range ticker.C {
			s := batcherState(tb.state.Load())
			if s != BStateRunning {
				return
			}
			tb.forceFlush()
		}
	}()
}

// forceFlush  不负责判断状态 它只是“把当前 buf 取出来并 flush” 状态控制由 startLoop 和 StopAndFlush 外部负责
func (tb *BaseBatcher) forceFlush() {
	var flushData []byte
	var flushBp *[]byte
	var count int

	tb.mtx.Lock()
	if tb.count > 0 && tb.buf != nil {
		flushData = *tb.buf
		flushBp = tb.buf
		count = tb.count
		tb.buf = nil
		tb.count = 0
		tb.wg.Add(1)
	}
	tb.mtx.Unlock()

	if count > 0 {
		tb.flushFn(flushData, flushBp, count)
		tb.wg.Done()
	}
}

// StopAndFlush 关闭并清空
func (tb *BaseBatcher) StopAndFlush() {
	if !tb.state.CompareAndSwap(int32(BStateRunning), int32(BStateClosing)) {
		return
	}

	tb.forceFlush()

	tb.wg.Wait()
	tb.state.Store(int32(BStateStopped))
}

// ------------------------actor 版本-------------------------
// 优点：Add锁小，并发快
// 缺点：异步持有了数据，外层不好做池化

// func (tb *PubBatcher) Add(ctx gctx.Context) error {
// 	err := tb.queue.Push(ctx)
// 	if err != nil {
// 		return err
// 	}
//
// 	return err
// }
//
// // 定时器：每 25 毫秒强制发送一次，防止消息一直积压不发
// func (tb *PubBatcher) startLoop() {
// 	const maxRetainBufSize = buffSize * 2
// 	tb.wg.Add(1)
// 	go func() {
// 		ticker := time.NewTicker(time.Millisecond * 25)
// 		defer func() {
// 			ticker.Stop()
// 			defer tb.wg.Done()
// 		}()
//
// 		buf := make([]byte, 0, buffSize)
// 		count := 0
//
// 		flush := func() {
// 			if count > 0 {
// 				err := tb.conn.Publish(tb.subject, buf)
// 				if err != nil {
// 					zap.L().Error("nats publish err", zap.Error(err))
// 				}
// 				if cap(buf) > maxRetainBufSize {
// 					buf = make([]byte, 0, buffSize)
// 				} else {
// 					buf = buf[:0]
// 				}
// 				count = 0
// 				tb.metricFlush.Add(count)
// 				tb.metricBatchSizeMsg.Update(float64(count))
// 			}
// 		}
//
// 		processFunc := func(ctx gctx.Context) {
// 			subSize := headerSize + len(ctx.Data)
//
// 			// 追加前缀
// 			buf = binary.LittleEndian.AppendUint32(buf, uint32(subSize))
//
// 			// 追加主体
// 			buf = binary.LittleEndian.AppendUint32(buf, ctx.MsgID)
// 			buf = binary.LittleEndian.AppendUint64(buf, ctx.ActorID)
// 			buf = binary.LittleEndian.AppendUint64(buf, ctx.SesID)
// 			buf = append(buf, ctx.FromSer, ctx.FromSerID, ctx.ToSer, ctx.ToSerID, ctx.Flag)
// 			buf = append(buf, ctx.Data...)
//
// 			count++
//
// 			// 如果在 Range 遍历期间达到了批处理上限，直接触发发送
// 			if count >= batchCount || len(buf) > buffSize {
// 				flush()
// 			}
// 		}
//
// 		for {
// 			select {
// 			case <-tb.queue.Sig():
// 				tb.queue.Range(processFunc)
// 			case <-ticker.C:
// 				tb.queue.Range(processFunc)
// 				flush()
// 			case <-tb.stopCh:
// 				tb.queue.Range(processFunc)
// 				flush()
// 				return
// 			}
// 		}
// 	}()
// }
