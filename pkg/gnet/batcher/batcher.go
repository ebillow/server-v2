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
	binary.LittleEndian.PutUint32(dst[0:], uint32(bodySize))
	binary.LittleEndian.PutUint32(dst[4:], ctx.MsgID)
	binary.LittleEndian.PutUint64(dst[8:], ctx.ActorID)
	binary.LittleEndian.PutUint64(dst[16:], ctx.SesID)
	dst[24] = ctx.FromSer
	dst[25] = ctx.FromSerID
	dst[26] = ctx.ToSer
	dst[27] = ctx.ToSerID
	dst[28] = ctx.Flag
	copy(dst[29:], ctx.Data)
}
