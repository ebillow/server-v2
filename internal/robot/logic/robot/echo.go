package robot

import (
	"server/api/pb"
	msgid2 "server/api/pb/msgid"
	clinet2 "server/internal/robot/clinet"
	"server/internal/robot/logic/monitor"
	"server/pkg/util"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func InitEcho(r *Robot) {
	clinet2.RegistryMsg(msgid2.MsgIDS2C_S2CEcho, func() proto.Message { return &pb.S2CEcho{} }, onEcho)
	r.AddTask(int64(util.RandRange(1, 1)), task)
	r.AddTask(int64(util.RandRange(1, 1)), task)
	r.AddTask(int64(util.RandRange(1, 1)), task)
	r.AddTask(int64(util.RandRange(1, 1)), task)
	r.AddTask(int64(util.RandRange(1, 1)), task)

}

func onEcho(msgBase proto.Message, ses *clinet2.Session) {
	msg := msgBase.(*pb.S2CEcho)
	r := ses.U.(*Robot)
	if msg.ID != r.Data.ID ||
		msg.Name != r.Data.Name ||
		msg.Exp != r.Data.Exp ||
		msg.Level != uint32(r.Data.Level) {
		zap.L().Warn("echo data not match", zap.Any("msg", msg), zap.Any("data", r.Data))
	}
	AddRecvCnt()
	monitor.Add(time.Since(time.Unix(0, msg.CliTime)))
}

func task(r *Robot) {
	r.Send(msgid2.MsgIDC2S_C2SEcho, &pb.C2SEcho{
		ID:    r.Data.ID,
		Name:  r.Data.Name,
		Level: uint32(r.Data.Level),
		Exp:   r.Data.Exp,
		Data:  "echo message test",
		Time:  time.Now().UnixNano(),
	})
	AddSendCnt()
}
