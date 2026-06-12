package msgq

import (
	"server/api/pb"
	"server/pkg/gerror"
	"server/pkg/gnet/dep"
	"server/pkg/gnet/pkg"
	"server/pkg/gnet/pub"
	"server/pkg/gnet/trace"
	"time"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

// RpcCall 远程调用，注意最好保持单向调用，否则可能出现互相等待
func RpcCall[Req pb.VTMessage, Res pb.VTMessage](bs *DataBus, req Req, res Res, toSer pb.Server, toSerID uint8, actorID uint64, timeOut time.Duration) error {
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

	bufPtr := pub.GetBuffer()
	defer pub.FreeBuffer(bufPtr)

	reqSize := req.SizeVT()
	bodySize := pkg.FrameBodyHeadSize + reqSize
	totalSize := bodySize + pkg.FrameLenSize

	if cap(*bufPtr) < totalSize {
		*bufPtr = make([]byte, totalSize)
	}
	buf := (*bufPtr)[:totalSize]

	head := pkg.Head{
		ActorID:   actorID,
		MsgID:     msgID,
		Flag:      0,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(toSer),
		ToSerID:   toSerID,
	}
	head.EncodeTo(buf, bodySize)

	bodyStart := pkg.FrameLenSize + pkg.FrameBodyHeadSize
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

	if trace.Rule.ShouldLog(msgID, actorID, 0) {
		reqStr, _ := sonic.MarshalString(req)
		resStr, _ := sonic.MarshalString(res)
		zap.L().Info("rpc call",
			zap.String("req", reqStr),
			zap.String("res", resStr),
			zap.Uint64("actorID", actorID),
			zap.String("to", pb.Server_name[int32(toSer)]),
			zap.Uint8("to", uint8(toSer)),
		)
	}

	return nil
}

func RpcRespond[T pb.VTMessage](bs *DataBus, reply string, ack T) error {
	if reply == "" {
		return nil
	}

	bufPtr := pub.GetBuffer()
	defer pub.FreeBuffer(bufPtr)

	size := ack.SizeVT()
	if cap(*bufPtr) < size {
		*bufPtr = make([]byte, size)
	}
	buf := (*bufPtr)[:size]

	n, err := ack.MarshalToSizedBufferVT(buf)
	if err != nil {
		return gerror.WithStack(err)
	}

	return bs.rpcConn.Publish(reply, buf[:n])
}
