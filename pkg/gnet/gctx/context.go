package gctx

import (
	"server/pkg/flag"
	"server/pkg/gnet/codec"
	"server/pkg/pb"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap/zapcore"
)

type Unity interface{}
type Context struct {
	U   Unity
	Raw *nats.Msg
	Msg *pb.NatsMsg
}

func (s Context) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	if s.Msg != nil {
		encoder.AddUint32("msgID", s.Msg.MsgID)
		encoder.AddUint64("roleID", s.Msg.RoleID)
		encoder.AddUint64("sesID", s.Msg.SesID)
		encoder.AddString("from", flag.SrvName(s.Msg.SerType))
		encoder.AddInt32("serID", s.Msg.SerID)
	}
	return nil
}

func (s Context) FreeNatsMsg() {
	codec.PutNatsMsg(s.Msg)
}
