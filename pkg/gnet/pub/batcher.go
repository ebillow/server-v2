package pub

import (
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gmsg"
	"sync"
	"sync/atomic"
	"time"
)

type batcherState int32

const (
	stateRunning batcherState = iota
	stateClosing
	stateStopped
)

// FlushFunc 是由具体的 NATS / JetStream 实现的发送回调
type FlushFunc func(data []byte, count int)

// Batcher 针对单个目标服务器的流式批处理器
type Batcher struct {
	mtx   sync.Mutex
	buf   *[]byte // 从 bufPool 获取的共享大缓冲
	count int     // 当前缓冲了多少条消息

	flushFn FlushFunc

	close     chan struct{}
	state     atomic.Int32
	wgTask    sync.WaitGroup
	wgLoop    sync.WaitGroup
	closeOnce sync.Once
}

func NewBatcher(flushFn FlushFunc) *Batcher {
	tb := &Batcher{
		flushFn: flushFn,
		close:   make(chan struct{}),
	}
	tb.wgLoop.Add(1)
	tb.startLoop()

	return tb
}

func (tb *Batcher) Add(p gmsg.Message) error {
	bodySize := gmsg.FrameBodyHeadSize + len(p.Data)
	frameSize := gmsg.FrameLenSize + bodySize
	var tasks [2]flushTask
	taskN := 0

	tb.mtx.Lock()

	if batcherState(tb.state.Load()) != stateRunning {
		tb.mtx.Unlock()
		return dep.ErrClosed
	}

	if tb.buf == nil {
		tb.buf = GetBuffer()
		*tb.buf = (*tb.buf)[:0]
	}

	if len(*tb.buf)+frameSize > cap(*tb.buf) && len(*tb.buf) > 0 {
		tasks[taskN] = flushTask{buf: *tb.buf, pBuf: tb.buf, count: tb.count}
		taskN++
		tb.buf = GetBuffer()
		*tb.buf = (*tb.buf)[:0]
		tb.count = 0
	}

	if frameSize > cap(*tb.buf) {
		monsterBuf := make([]byte, frameSize)
		p.EncodeTo(monsterBuf, bodySize)

		// bp 传 nil，表示不需要 FreeBuffer
		tasks[taskN] = flushTask{buf: monsterBuf, pBuf: nil, count: 1}
		taskN++
	} else {
		buf := *tb.buf
		pos := len(buf)
		buf = buf[:pos+frameSize]

		p.EncodeTo(buf[pos:pos+frameSize], bodySize)

		*tb.buf = buf
		tb.count++

		if tb.count >= batchCount || len(*tb.buf) >= buffSize {
			tasks[taskN] = flushTask{buf: *tb.buf, pBuf: tb.buf, count: tb.count}
			taskN++
			tb.buf = nil // 置空，下一次循环会重新 GetBuffer
			tb.count = 0
		}
	}

	if taskN > 0 {
		tb.wgTask.Add(taskN)
	}
	tb.mtx.Unlock()

	for i := 0; i < taskN; i++ {
		t := tasks[i]
		tb.flushFn(t.buf, t.count)
		if t.pBuf != nil {
			FreeBuffer(t.pBuf)
		}
		tb.wgTask.Done()
	}

	return nil
}

type flushTask struct {
	buf   []byte
	pBuf  *[]byte // 如果是 nil，说明是临时大包，不需要回收到 pool
	count int
}

// 定时器：每 25 毫秒强制发送一次，防止消息一直积压不发
func (tb *Batcher) startLoop() {
	go func() {
		ticker := time.NewTicker(time.Millisecond * 25)
		defer func() {
			ticker.Stop()
			tb.wgLoop.Done()
		}()

		for {
			select {
			case <-ticker.C:
				tb.flush()
			case <-tb.close:
				tb.flush()
				return
			}
		}
	}()
}

// flush  不负责判断状态 它只是“把当前 buf 取出来并 flush” 状态控制由 startLoop 和 Close 外部负责
func (tb *Batcher) flush() {
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
		tb.wgTask.Add(1)
	}
	tb.mtx.Unlock()

	if count > 0 {
		tb.flushFn(flushData, count)
		FreeBuffer(flushBp)
		tb.wgTask.Done()
	}
}

// Close 关闭并清空
func (tb *Batcher) Close() {
	tb.closeOnce.Do(func() {
		tb.state.Store(int32(stateClosing))
		close(tb.close)
		tb.wgLoop.Wait()
		tb.wgTask.Wait()
		tb.state.Store(int32(stateStopped))
	})
}
