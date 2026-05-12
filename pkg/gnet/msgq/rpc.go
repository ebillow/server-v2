package msgq

import (
	"encoding/binary"
	"server/pkg/gerror"
	"server/pkg/pb"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// RpcCall 远程调用，注意最好保持单向调用，否则可能出现互相等待
func RpcCall(bs *DataBus, msgID uint32, req proto.Message, ack proto.Message, toSer pb.Server, toSerID int32, roleID uint64, sesID uint64, timeOut time.Duration) error {
	bufPtr := GetBuffer()
	defer FreeBuffer(bufPtr)

	buf := (*bufPtr)[:0]

	reqSize := proto.Size(req)
	subSize := headerSize + reqSize

	buf = binary.LittleEndian.AppendUint32(buf, uint32(subSize))
	buf = binary.LittleEndian.AppendUint32(buf, msgID)
	buf = binary.LittleEndian.AppendUint64(buf, roleID)
	buf = binary.LittleEndian.AppendUint64(buf, sesID)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(bs.serType))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(bs.serID))
	buf = append(buf, 0)

	subStr := getRpcIdxSubject(toSer, toSerID)

	marshalOpts := proto.MarshalOptions{}
	var err error
	buf, err = marshalOpts.MarshalAppend(buf, req)
	if err != nil {
		return gerror.Wrapf(err, "rpc call:marshal err; msg[%d] to %v %d", msgID, toSer, toSerID)
	}

	resMsg, err := bs.rpcConn.Request(subStr, buf, timeOut)
	if err != nil {
		return gerror.Wrapf(err, "rpc call:request err; msg[%d] to %v %d", msgID, toSer, toSerID)
	}

	err = proto.Unmarshal(resMsg.Data, ack)
	if err != nil {
		return gerror.Wrapf(err, "rpc call:unmarshal err; msg[%d] to %v %d", msgID, toSer, toSerID)
	}

	return nil
}

func RpcRespond(msg *nats.Msg, ack proto.Message) error {
	bufPtr := GetBuffer()
	defer FreeBuffer(bufPtr)

	buf := (*bufPtr)[:0]
	marshalOpts := proto.MarshalOptions{}
	buf, err := marshalOpts.MarshalAppend(buf, ack)
	if err != nil {
		return gerror.WithStack(err)
	}
	return msg.Respond(buf)
}
