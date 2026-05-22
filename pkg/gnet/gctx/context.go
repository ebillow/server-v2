package gctx

import (
	"server/api/pb"
	"server/pkg/flag"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap/zapcore"
)

const (
	Normal    = 0
	Forward   = 1
	Broadcast = 1 << 1
)

type Unity interface{}
type Context struct {
	Data []byte
	U    Unity
	Raw  *nats.Msg

	ActorID   uint64
	SesID     uint64
	MsgID     uint32
	Flag      uint8
	FromSer   uint8
	FromSerID uint8
	ToSer     uint8
	ToSerID   uint8
}

func (s *Context) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint32("msgID", s.MsgID)
	encoder.AddUint64("actorID", s.ActorID)
	encoder.AddUint64("sesID", s.SesID)
	encoder.AddString("from", flag.SrvName(pb.Server(s.FromSer)))
	encoder.AddUint8("fromID", s.FromSerID)
	encoder.AddString("to", flag.SrvName(pb.Server(s.ToSer)))
	encoder.AddUint8("toID", s.ToSerID)
	return nil
}
