package batcher

import (
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gctx"
	"sync"
	"sync/atomic"
	"time"
)

// BaseBatcher 针对单个目标服务器的流式批处理器
type BaseBatcher struct {
	mtx   sync.Mutex
	buf   *[]byte // 从 bufPool 获取的共享大缓冲
	count int     // 当前缓冲了多少条消息
	state atomic.Int32
	wg    sync.WaitGroup

	flushFn FlushFunc
}

func NewBaseBatcher(flushFn FlushFunc) *BaseBatcher {
	tb := &BaseBatcher{
		flushFn: flushFn,
	}
	tb.state.Store(int32(BStateRunning))
	tb.startLoop()

	return tb
}

func (tb *BaseBatcher) Add(ctx gctx.Context) error {
	bodySize := gctx.FrameBodyHeadSize + len(ctx.Data)
	frameSize := gctx.FrameLenSize + bodySize
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
		gctx.SerializeFrame(monsterBuf, bodySize, ctx)

		// bp 传 nil，表示不需要 FreeBuffer
		tasks[taskN] = flushTask{data: monsterBuf, bp: nil, count: 1}
		taskN++
	} else {
		buf := *tb.buf
		pos := len(buf)
		buf = buf[:pos+frameSize]

		gctx.SerializeFrame(buf[pos:pos+frameSize], bodySize, ctx)

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
		tb.wg.Add(taskN)
	}
	tb.mtx.Unlock()

	for i := 0; i < taskN; i++ {
		t := tasks[i]
		tb.flushFn(t.data, t.count)
		if t.bp != nil {
			FreeBuffer(t.bp)
		}
		tb.wg.Done()
	}

	return nil
}

type flushTask struct {
	data  []byte
	bp    *[]byte // 如果是 nil，说明是临时大包，不需要回收到 pool
	count int
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
		tb.flushFn(flushData, count)
		FreeBuffer(flushBp)
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
