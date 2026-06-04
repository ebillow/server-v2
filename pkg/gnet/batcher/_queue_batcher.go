package batcher

import (
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gctx"
	"server/pkg/queue"
	"sync"
	"sync/atomic"
	"time"
)

// 现在gateway做了池化，不要用这个方案

// QueueBatcher  优点：Add锁小，并发快
// 缺点：异步持有了数据，外层不好做池化,
type QueueBatcher struct {
	queue   *queue.SwapQueue[gctx.Context]
	state   atomic.Int32
	flushFn FlushFunc
	wg      sync.WaitGroup
}

func NewQueueBatcher(flushFn FlushFunc) *QueueBatcher {
	tb := &QueueBatcher{
		queue:   queue.NewSwapQueue[gctx.Context](4096, 40960),
		flushFn: flushFn,
	}

	tb.startLoop()

	return tb
}

func (tb *QueueBatcher) Add(ctx gctx.Context) error {
	if batcherState(tb.state.Load()) != BStateRunning {
		return dep.ErrClosed
	}
	err := tb.queue.Push(ctx)
	if err != nil {
		return err
	}

	return err
}

func (tb *QueueBatcher) startLoop() {
	tb.wg.Add(1)
	go func() {
		ticker := time.NewTicker(time.Millisecond * 25)
		defer func() {
			tb.wg.Done()
			ticker.Stop()
		}()

		buf := make([]byte, 0, buffSize)
		count := 0

		flush := func() {
			if count > 0 {
				tb.flushFn(buf, count)
				if cap(buf) > buffSize {
					buf = make([]byte, 0, buffSize)
				} else {
					buf = buf[:0]
				}
				count = 0
			}
		}

		processFunc := func(ctx gctx.Context) {
			msgSize := FrameBodyHeadSize + len(ctx.Data)
			frameSize := 4 + msgSize
			if len(buf)+frameSize > cap(buf) && len(buf) > 0 {
				flush()
			}
			if frameSize > cap(buf) {
				monsterBuf := make([]byte, frameSize)
				SerializeFrame(monsterBuf, msgSize, ctx)
				tb.flushFn(monsterBuf, 1)
			} else {
				pos := len(buf)
				buf = buf[:pos+frameSize]
				SerializeFrame(buf[pos:pos+frameSize], msgSize, ctx)
				count++
				// 如果在 Range 遍历期间达到了批处理上限，直接触发发送
				if count >= batchCount || len(buf) > buffSize {
					flush()
				}
			}
		}

		for {
			select {
			case <-tb.queue.Sig():
				tb.queue.Range(processFunc)

			case <-ticker.C:
				tb.queue.Range(processFunc)
				flush()
			}

			if tb.queue.IsClosed() {
				tb.queue.Range(processFunc)
				flush()
				return
			}
		}
	}()
}

// StopAndFlush 关闭并清空
func (tb *QueueBatcher) StopAndFlush() {
	if !tb.state.CompareAndSwap(int32(BStateRunning), int32(BStateClosing)) {
		return
	}
	tb.wg.Wait()
	tb.state.Store(int32(BStateStopped))
}
