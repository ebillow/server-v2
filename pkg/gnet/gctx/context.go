package gctx

import (
	"server/pkg/flag"
	"server/pkg/pb"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap/zapcore"
)

type Unity interface{}
type Context struct {
	Data []byte
	U    Unity
	Raw  *nats.Msg

	RoleID  uint64
	SesID   uint64
	MsgID   uint32
	SerID   int32
	SerType pb.Server
	Forward uint8
}

func (s Context) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint32("msgID", s.MsgID)
	encoder.AddUint64("roleID", s.RoleID)
	encoder.AddUint64("sesID", s.SesID)
	encoder.AddString("from", flag.SrvName(s.SerType))
	encoder.AddInt32("serID", s.SerID)

	return nil
}
