package codec

import (
	"google.golang.org/protobuf/proto"
	"server/pkg/pb"
	"sync"
)

// bufPool 序列化 Buffer 对象池
//
//	预分配 512 字节的容量，减少扩容次数。可根据游戏实际包体大小调整。
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 512)
		return &b
	},
}

// FreeBuffer 发送完毕后，需要手动归还 []byte
func FreeBuffer(b []byte) {
	// 防御性编程：容量过大的异常包直接丢弃，防止占用过多常驻内存
	if cap(b) > 64*1024 {
		return
	}
	bufPool.Put(&b)
}

// NatsMsgPool NatsMsg 对象池
var NatsMsgPool = sync.Pool{
	New: func() any {
		return &pb.NatsMsg{}
	},
}

func GetNatsMsg() *pb.NatsMsg {
	return NatsMsgPool.Get().(*pb.NatsMsg)
}

func PutNatsMsg(msg *pb.NatsMsg) {
	if msg == nil {
		return
	}
	// 【致命关键】放回池子前必须 Reset，否则会带有上一次的脏数据（尤其是 slice 字段）
	proto.Reset(msg)
	NatsMsgPool.Put(msg)
}
