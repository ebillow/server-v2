package codec

import (
	"google.golang.org/protobuf/proto"
	"server/pkg/pb"
)

func Encode(msg *pb.NatsMsg) ([]byte, error) {
	bp := bufPool.Get().(*[]byte)
	*bp = (*bp)[:0] // 重置长度为 0，但保留 capacity

	// 使用 MarshalAppend 复用底层数组
	mo := proto.MarshalOptions{}
	b, err := mo.MarshalAppend(*bp, msg)
	if err != nil {
		bufPool.Put(bp) // 出错也要归还
		return nil, err
	}

	return b, nil
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
