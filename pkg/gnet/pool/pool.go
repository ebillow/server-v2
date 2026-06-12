package pool

import (
	"sync"

	"google.golang.org/protobuf/proto"
)

var (
	BufPool512  = NewBytesPoll(512)
	BufPool1024 = NewBytesPoll(1024)
	BufPool5120 = NewBytesPoll(5120)
)

type BytesPool struct {
	pool sync.Pool
	size int
}

func NewBytesPoll(buffSize int) *BytesPool {
	return &BytesPool{
		size: buffSize,
		pool: sync.Pool{
			New: func() any {
				b := make([]byte, 0, buffSize)
				return &b
			},
		},
	}
}

func (p *BytesPool) Get() *[]byte {
	return p.pool.Get().(*[]byte)
}

func (p *BytesPool) Put(b *[]byte) {
	//  容量过大的异常包直接丢弃，防止占用过多常驻内存
	if cap(*b) > 10*p.size {
		return
	}
	*b = (*b)[:0] // 重置长度为 0，但保留 capacity
	p.pool.Put(b)
}

type MsgPool[T any, PT interface {
	*T
	proto.Message
}] struct {
	pool sync.Pool
}

func NewMsgPool[T any, PT interface {
	*T
	proto.Message
}]() *MsgPool[T, PT] {
	return &MsgPool[T, PT]{
		pool: sync.Pool{
			New: func() any {
				// new(T) 会真正分配底层结构体的内存，并返回 *pb.User
				return PT(new(T))
			},
		},
	}
}

func (p *MsgPool[T, PT]) Get() PT {
	msg := p.pool.Get().(PT)
	proto.Reset(msg) // 此时 msg 是 PT 类型
	return msg
}

func (p *MsgPool[T, PT]) Put(msg PT) {
	p.pool.Put(msg)
}
