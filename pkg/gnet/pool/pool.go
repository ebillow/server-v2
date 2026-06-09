package pool

import "sync"

var (
	BufPool512  = NewBytesPoll(512)
	BufPool1024 = NewBytesPoll(1024)
	BufPool5120 = NewBytesPoll(5120)
)

type BytesPool struct {
	sync.Pool
	size int
}

func NewBytesPoll(size int) *BytesPool {
	return &BytesPool{
		size: size,
		Pool: sync.Pool{
			New: func() any {
				b := make([]byte, 0, size)
				return &b
			},
		},
	}
}

func (p *BytesPool) GetBuffer() *[]byte {
	return p.Get().(*[]byte)
}

func (p *BytesPool) PutBuffer(b *[]byte) {
	//  容量过大的异常包直接丢弃，防止占用过多常驻内存
	if cap(*b) > 10*p.size {
		return
	}
	*b = (*b)[:0] // 重置长度为 0，但保留 capacity
	p.Put(b)
}
