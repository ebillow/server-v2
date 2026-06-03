package robot

import (
	"server/api/pb"
	"server/api/pb/msgid"
	clinet2 "server/internal/robot/clinet"
	"server/pkg/util"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const TimeOutMs = 300

// RegisteMsgHandle 注册消息处理函数
func RegisteMsgHandle() {
	clinet2.RegistryMsg(msgid.MsgIDS2C_S2CLogin, func() proto.Message { return &pb.S2CLogin{} }, onLogin)
	clinet2.RegistryMsg(msgid.MsgIDS2C_S2CHeartBeat, func() proto.Message { return &pb.S2CHeartBeat{} }, onHeartbeat)
}

func onLogin(msgBase proto.Message, ses *clinet2.Session) {
	msg := msgBase.(*pb.S2CLogin)
	r := ses.U.(*Robot)

	if msg.Code != pb.LoginCode_LCSuccess {
		zap.S().Warnf("%s login err:%s", r.acc, pb.LoginCode_name[int32(msg.Code)])
		return
	}

	r.onLoginSuccess(msg)
}

func onHeartbeat(msgBase proto.Message, ses *clinet2.Session) {
	msg := msgBase.(*pb.S2CHeartBeat)

	r := ses.U.(*Robot)
	if r == nil {
		return
	}

	// fmt.Printf("game time out :%ds", util.GetNowTimeM()-msg.CliTime)

	if util.GetNowTimeM()-msg.CliTime > TimeOutMs {
		TimeOut(r.id)
	}
	if r.Data != nil {
		Active(r.Data.ID)
	}
}
