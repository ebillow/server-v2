package msgq

import (
	"server/api/pb"
	"server/pkg/gerror"
	"server/pkg/gnet/batcher"
	"server/pkg/gnet/dep"
	"time"

	"github.com/nats-io/nats.go"
)

// RpcCall 远程调用，注意最好保持单向调用，否则可能出现互相等待
func RpcCall[Req pb.VTMessage, Res pb.VTMessage](bs *DataBus, req Req, res Res, toSer pb.Server, toSerID uint8, actorID uint64, sesID uint64, timeOut time.Duration) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}
	if bs.rpcConn == nil {
		return gerror.New("rpc call: rpc conn nil")
	}

	msgID, ok := pb.GetMsgIDS2S(req) // 假设你之前的 TypeMeta 是全局/能访问的
	if !ok {
		return gerror.Newf("rpc call: msg id not found for type %T", req)
	}

	bufPtr := batcher.GetBuffer()
	defer batcher.FreeBuffer(bufPtr)

	reqSize := req.SizeVT()
	bodySize := batcher.FrameBodyHeadSize + reqSize
	totalSize := bodySize + batcher.FrameLenSize

	if cap(*bufPtr) < totalSize {
		*bufPtr = make([]byte, totalSize)
	}
	buf := (*bufPtr)[:totalSize]

	batcher.WriteFrameHeader(buf, bodySize, msgID, actorID, sesID, bs.serType, bs.serID, uint8(toSer), toSerID, 0)

	bodyStart := batcher.FrameLenSize + batcher.FrameBodyHeadSize
	n, err := req.MarshalToSizedBufferVT(buf[bodyStart:])
	if err != nil {
		return gerror.Wrapf(err, "rpc call:marshal err; msg[%d] to %v %d", msgID, toSer, toSerID)
	}

	buf = buf[:bodyStart+n]

	subStr := rpcIdxSubjectName(toSer, toSerID)
	if timeOut <= 0 {
		timeOut = time.Second
	}
	resMsg, err := bs.rpcConn.Request(subStr, buf, timeOut)
	if err != nil {
		return gerror.Wrapf(err, "rpc call:request err; msg[%d] to %v %d", msgID, toSer, toSerID)
	}

	err = res.UnmarshalVT(resMsg.Data)
	if err != nil {
		return gerror.Wrapf(err, "rpc call:unmarshal err; msg[%d] to %v %d", msgID, toSer, toSerID)
	}

	return nil
}

func RpcRespond[T pb.VTMessage](msg *nats.Msg, ack T) error {
	bufPtr := batcher.GetBuffer()
	defer batcher.FreeBuffer(bufPtr)

	size := ack.SizeVT()
	if cap(*bufPtr) < size {
		*bufPtr = make([]byte, size)
	}
	buf := (*bufPtr)[:size]

	n, err := ack.MarshalToSizedBufferVT(buf)
	if err != nil {
		return gerror.WithStack(err)
	}

	return msg.Respond(buf[:n])
}
