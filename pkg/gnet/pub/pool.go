package pub

import "server/pkg/gnet/pool"

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
