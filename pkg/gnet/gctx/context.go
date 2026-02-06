package gctx

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap/zapcore"
	"server/pkg/flag"
	"server/pkg/pb"
)

type Unity interface{}
type Context struct {
	U       Unity
	Raw     *nats.Msg
	Msg     *pb.NatsMsg
	MsgName map[int32]string
}

func (s Context) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint32("msgID", s.Msg.MsgID)
	encoder.AddString("msgName", s.MsgName[int32(s.Msg.MsgID)])
	encoder.AddUint64("roleID", s.Msg.RoleID)
	encoder.AddUint64("sesID", s.Msg.SesID)
	encoder.AddString("from", flag.SrvName(s.Msg.SerType))
	encoder.AddInt32("serID", s.Msg.SerID)
	return nil
}
