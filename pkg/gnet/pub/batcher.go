package pub

import "server/pkg/gnet/pool"

type batcherState int32

const (
	BStateRunning batcherState = iota
	BStateClosing
	BStateStopped
)

const (
	batchCount = 500
	buffSize   = 1024 * batchCount
)

var bufPool = pool.NewBytesPoll(buffSize)

func GetBuffer() *[]byte {
	return bufPool.Get()
}

func FreeBuffer(b *[]byte) {
	bufPool.Put(b)
}

// FlushFunc 是由具体的 NATS / JetStream 实现的发送回调
type FlushFunc func(data []byte, count int)
