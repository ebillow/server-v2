package msgq

import "sync"

// bufPool 序列化 Buffer 对象池
//
//	预分配
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, buffSize)
		return &b
	},
}

func GetBuffer() *[]byte {
	return bufPool.Get().(*[]byte)
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
