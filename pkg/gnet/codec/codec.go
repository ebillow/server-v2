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
func FreeBuffer(b *[]byte) {
	//  容量过大的异常包直接丢弃，防止占用过多常驻内存
	if cap(*b) > 64*1024 {
		return
	}
	*b = (*b)[:0] // 重置长度为 0，但保留 capacity
	bufPool.Put(b)
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
	// 放回池子前必须 Reset，否则会带有上一次的脏数据（尤其是 slice 字段）
	proto.Reset(msg)
	NatsMsgPool.Put(msg)
}

func Encode(msg *pb.NatsMsg) ([]byte, *[]byte, error) {
	bp := bufPool.Get().(*[]byte)

	// 使用 MarshalAppend 复用底层数组
	mo := proto.MarshalOptions{}
	b, err := mo.MarshalAppend(*bp, msg)
	if err != nil {
		FreeBuffer(bp) // 出错也要归还
		return nil, bp, err
	}

	return b, bp, nil
}

func Decode(in []byte) (*pb.NatsMsg, error) {
	msg := GetNatsMsg()
	err := proto.Unmarshal(in, msg)
	if err != nil {
		PutNatsMsg(msg) // 解码失败，立即回收
		return nil, err
	}
	return msg, nil
}

//
// const headerSize = 4 + 8 + 8 + 4 + 4 + 1
//
// func Encode2(msgID uint32, data []byte, roleID uint64, sesID uint64, serTyp pb.Server, serID int32, forward uint8) ([]byte, *[]byte, error) {
// 	bp := bufPool.Get().(*[]byte)
// 	buf := (*bp)[:0]
//
// 	totalSize := headerSize + len(data)
// 	if cap(buf) < totalSize {
// 		buf = make([]byte, 0, totalSize)
// 	}
//
// 	buf = binary.LittleEndian.AppendUint32(buf, msgID)
// 	buf = binary.LittleEndian.AppendUint64(buf, roleID)
// 	buf = binary.LittleEndian.AppendUint64(buf, sesID)
// 	buf = binary.LittleEndian.AppendUint32(buf, uint32(serTyp))
// 	buf = binary.LittleEndian.AppendUint32(buf, uint32(serID))
// 	buf = append(buf, forward)
// 	buf = append(buf, data...)
//
// 	*bp = buf
//
// 	return buf, bp, nil
// }
// func Decode2(buf []byte) (msgID uint32, data []byte, roleID uint64, sesID uint64, serTyp pb.Server, serID int32, forward uint8, err error) {
// 	if len(buf) < headerSize {
// 		return 0, nil, 0, 0, 0, 0, 0, errors.New("decode error: buffer too small for header")
// 	}
//
// 	offset := 0
//
// 	msgID = binary.LittleEndian.Uint32(buf[offset:])
// 	offset += 4
//
// 	roleID = binary.LittleEndian.Uint64(buf[offset:])
// 	offset += 8
//
// 	sesID = binary.LittleEndian.Uint64(buf[offset:])
// 	offset += 8
//
// 	serTyp = pb.Server(binary.LittleEndian.Uint32(buf[offset:]))
// 	offset += 4
//
// 	serID = int32(binary.LittleEndian.Uint32(buf[offset:]))
// 	offset += 4
//
// 	forward = buf[offset]
// 	offset += 1
//
// 	data = buf[offset:]
//
// 	return msgID, data, roleID, sesID, serTyp, serID, forward, nil
// }
