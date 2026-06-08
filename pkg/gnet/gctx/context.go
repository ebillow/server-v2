package gctx

import (
	"encoding/binary"
	"server/api/pb"
	"server/pkg/flag"
	"server/pkg/gerror"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap/zapcore"
)

const (
	Normal    = 0
	Forward   = 1
	Broadcast = 1 << 1
)

const FrameLenSize = 4
const FrameBodyHeadSize = 4 + 8 + 8 + 1 + 1 + 1 + 1 + 1

type Unity interface{}

type Head struct {
	ActorID   uint64
	SesID     uint64
	MsgID     uint32
	Flag      uint8
	FromSer   uint8
	FromSerID uint8
	ToSer     uint8
	ToSerID   uint8
}

type Context struct {
	Head Head
	Data []byte // 逻辑层不能持有
	U    Unity
	Raw  *nats.Msg // 逻辑层不能持有
}

func (s *Context) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint32("msgID", s.Head.MsgID)
	encoder.AddUint64("actorID", s.Head.ActorID)
	encoder.AddUint64("sesID", s.Head.SesID)
	encoder.AddString("from", flag.SrvName(pb.Server(s.Head.FromSer)))
	encoder.AddUint8("fromID", s.Head.FromSerID)
	encoder.AddString("to", flag.SrvName(pb.Server(s.Head.ToSer)))
	encoder.AddUint8("toID", s.Head.ToSerID)
	return nil
}

func SerializeFrame(dst []byte, bodySize int, ctx Context) {
	EncodeFrameHeader(dst, bodySize, ctx.Head)
	copy(dst[FrameLenSize+FrameBodyHeadSize:], ctx.Data)
}

func EncodeFrameHeader(dst []byte, bodySize int, head Head) {
	binary.LittleEndian.PutUint32(dst[0:], uint32(bodySize))
	binary.LittleEndian.PutUint32(dst[4:], head.MsgID)
	binary.LittleEndian.PutUint64(dst[8:], head.ActorID)
	binary.LittleEndian.PutUint64(dst[16:], head.SesID)
	dst[24] = head.FromSer
	dst[25] = head.FromSerID
	dst[26] = head.ToSer
	dst[27] = head.ToSerID
	dst[28] = head.Flag
}

func DecodeHead(buf []byte) (head Head, err error) {
	if len(buf) < FrameBodyHeadSize {
		return head, gerror.New("decode error: buffer too small for header")
	}

	offset := 0

	head.MsgID = binary.LittleEndian.Uint32(buf[offset:])
	offset += 4

	head.ActorID = binary.LittleEndian.Uint64(buf[offset:])
	offset += 8

	head.SesID = binary.LittleEndian.Uint64(buf[offset:])
	offset += 8

	head.FromSer = buf[offset]
	offset += 1

	head.FromSerID = buf[offset]
	offset += 1

	head.ToSer = buf[offset]
	offset += 1

	head.ToSerID = buf[offset]
	offset += 1

	head.Flag = buf[offset]

	return head, nil
}
