package msgq

import (
	"server/pkg/gerror"
	"server/pkg/gnet/codec"
	"server/pkg/pb"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// RpcCall 远程调用，注意最好保持单向调用，否则可能出现互相等待
func RpcCall[T any, PT interface {
	*T
	proto.Message
}](bs DataBus, msgID uint32, req proto.Message, toSer pb.Server, toSerID int32, roleID uint64, sesID uint64, timeOut time.Duration) (res PT, err error) {
	ack := PT(new(T))
	toSub := getIndexSubject(toSer, toSerID)
	b, err := proto.Marshal(req)
	if err != nil {
		return ack, gerror.Wrapf(err, "rpc call:marshal err; msg[%d] to %s", msgID, toSub)
	}

	out, bp, err := codec.Encode(&pb.NatsMsg{
		MsgID:   msgID,
		Data:    b,
		SerID:   bs.serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: false,
	})
	resMsg, err := bs.conn.Request(toSub, out, timeOut)
	if err != nil {
		codec.FreeBuffer(bp)
		return ack, gerror.Wrapf(err, "rpc call:request err; msg[%d]", msgID)
	}
	codec.FreeBuffer(bp)

	// nstMsg, err := codec.Decode(resMsg.Data)
	// if err != nil {
	// 	return ack, gerror.Wrapf(err, "rpc call:unmarshal err; msg[%d]", msgID)
	// }
	err = proto.Unmarshal(resMsg.Data, ack)
	if err != nil {
		return ack, gerror.Wrapf(err, "rpc call:unmarshal err; msg[%d]", msgID)
	}
	// codec.PutNatsMsg(nstMsg)

	return ack, nil
}

func RpcRespond(msg *nats.Msg, ack proto.Message) error {
	b, err := proto.Marshal(ack)
	if err != nil {
		return gerror.WithStack(err)
	}
	return msg.Respond(b)
}
