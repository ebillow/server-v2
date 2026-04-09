package msgq

import (
	"server/pkg/gerror"
	"server/pkg/gnet/codec"
	"server/pkg/pb"
	"time"

	"google.golang.org/protobuf/proto"
)

func RpcCall[T proto.Message](bs DataBus, msgID uint32, req proto.Message, toSer pb.Server, toSerID int32, roleID uint64, sesID uint64, timeOut time.Duration) (res T, err error) {
	var ack T
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

	nstMsg, err := codec.Decode(resMsg.Data)
	if err != nil {
		return ack, gerror.Wrapf(err, "rpc call:unmarshal err; msg[%d]", msgID)
	}
	err = proto.Unmarshal(nstMsg.Data, ack)
	if err != nil {
		return ack, gerror.Wrapf(err, "rpc call:unmarshal err; msg[%d]", msgID)
	}
	codec.PutNatsMsg(nstMsg)

	return ack, nil
}
