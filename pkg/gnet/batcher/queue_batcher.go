package batcher

import (
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gctx"
	"server/pkg/queue"
	"sync/atomic"
	"time"
)

// QueueBatcher  优点：Add锁小，并发快
// 缺点：异步持有了数据，外层不好做池化
type QueueBatcher struct {
	queue   *queue.SwapQueue[gctx.Context]
	state   atomic.Int32
	flushFn FlushFunc
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
	const maxRetainBufSize = buffSize * 2
	go func() {
		ticker := time.NewTicker(time.Millisecond * 25)
		defer func() {
			ticker.Stop()
		}()

		buf := make([]byte, 0, buffSize)
		count := 0

		flush := func() {
			if count > 0 {
				tb.flushFn(buf, count)
				if cap(buf) > maxRetainBufSize {
					buf = make([]byte, 0, buffSize)
				} else {
					buf = buf[:0]
				}
				count = 0
			}
		}

		processFunc := func(ctx gctx.Context) {
			msgSize := HeaderSize + len(ctx.Data)
			frameSize := 4 + msgSize
			if len(buf)+frameSize > cap(buf) && len(buf) > 0 {
				flush()
			}
			if frameSize > cap(buf) {
				monsterBuf := make([]byte, frameSize)
				SerializeFrame(monsterBuf, msgSize, ctx)
				flush()
			} else {
				SerializeFrame(buf, msgSize, ctx)
				count++
				// 如果在 Range 遍历期间达到了批处理上限，直接触发发送
				if count >= batchCount || len(buf) > buffSize {
					flush()
				}
			}
		}

		for {
			select {
			case ok := <-tb.queue.Sig():
				tb.queue.Range(processFunc)
				if !ok {
					flush()
					return
				}
			case <-ticker.C:
				tb.queue.Range(processFunc)
				flush()
			}
		}
	}()
}

// StopAndFlush 关闭并清空
func (tb *QueueBatcher) StopAndFlush() {
	if !tb.state.CompareAndSwap(int32(BStateRunning), int32(BStateClosing)) {
		return
	}

	tb.queue.Close()
	tb.state.Store(int32(BStateStopped))
}
