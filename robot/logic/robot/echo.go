package robot

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"server/pkg/util"
	"server/robot/clinet"
)

func InitEcho(r *Robot) {
	clinet.RegistryMsg(msgid.MsgIDS2C_S2CEcho, func() proto.Message { return &pb.S2CEcho{} }, onProto)
	r.AddTask(int64(util.RandRangeFloat(1, 1)), task)
}

func onProto(msgBase proto.Message, ses *clinet.Session) {
	msg := msgBase.(*pb.S2CEcho)
	r := ses.U.(*Robot)
	if msg.ID != r.Data.ID ||
		msg.Name != r.Data.Name ||
		msg.Exp != r.Data.Exp ||
		msg.Level != uint32(r.Data.Level) {
		zap.L().Warn("echo data not match", zap.Any("msg", msg), zap.Any("data", r.Data))
	}
}

func task(r *Robot) {
	r.Send(msgid.MsgIDC2S_C2SEcho, &pb.C2SEcho{
		ID:    r.Data.ID,
		Name:  r.Data.Name,
		Level: uint32(r.Data.Level),
		Exp:   r.Data.Exp,
		Data:  "echo message test",
	})
}
