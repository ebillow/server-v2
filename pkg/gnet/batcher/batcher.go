package batcher

import (
	"encoding/binary"
	"server/pkg/gnet/gctx"
)

type batcherState int32

const (
	BStateRunning batcherState = iota
	BStateClosing
	BStateStopped
)

// FlushFunc 是由具体的 NATS / JetStream 实现的发送回调
type FlushFunc func(data []byte, count int)

func SerializeFrame(dst []byte, bodySize int, ctx gctx.Context) {
	WriteFrameHeader(dst,
		bodySize,
		ctx.MsgID,
		ctx.ActorID,
		ctx.SesID,
		ctx.FromSer,
		ctx.FromSerID,
		ctx.ToSer,
		ctx.ToSerID,
		ctx.Flag)
	copy(dst[FrameLenSize+FrameBodyHeadSize:], ctx.Data)
}
func WriteFrameHeader(
	dst []byte,
	bodySize int,
	msgID uint32,
	actorID uint64,
	sesID uint64,
	fromSer uint8,
	fromSerID uint8,
	toSer uint8,
	toSerID uint8,
	flag uint8,
) {
	binary.LittleEndian.PutUint32(dst[0:], uint32(bodySize))
	binary.LittleEndian.PutUint32(dst[4:], msgID)
	binary.LittleEndian.PutUint64(dst[8:], actorID)
	binary.LittleEndian.PutUint64(dst[16:], sesID)
	dst[24] = fromSer
	dst[25] = fromSerID
	dst[26] = toSer
	dst[27] = toSerID
	dst[28] = flag
}
