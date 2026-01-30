package msgq

import (
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"server/pkg/flag"
	"server/pkg/pb"
	"time"
)

func RpcCall[T proto.Message](bs DataBus, msgID uint32, req proto.Message, toSer pb.Server, toSerID int32, roleID uint64, sesID uint64) (res T, err error) {
	var ack T
	toSub := getIndexSubject(flag.SrvName(toSer), toSerID)
	b, err := proto.Marshal(req)
	if err != nil {
		return ack, errors.Wrapf(err, "rpc call:marshal err; msg[%d] to %s", msgID, toSub)
	}
	msg := nats.NewMsg(toSub)
	bs.setHeader(msg, msgID, roleID, sesID)

	msg.Data = b
	resMsg, err := bs.conn.RequestMsg(msg, time.Second*3)
	if err != nil {
		return ack, errors.Wrapf(err, "rpc call:request err; msg[%d]", msgID)
	}

	err = proto.Unmarshal(resMsg.Data, ack)
	if err != nil {
		return ack, errors.Wrapf(err, "rpc call:unmarshal err; msg[%d]", msgID)
	}

	return ack, nil
}
