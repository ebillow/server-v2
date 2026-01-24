package gnet

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/internal/gnet/msgq"
	"server/internal/pb"
)

func SendToRole(sesID uint64, msg proto.Message) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zap.L().Warn("send to role error", zap.Error(err))
		return
	}
	msgID, err := pb.GetMsgIDS2C(msg)
	if err != nil {
		zap.L().Warn("send to role error", zap.Error(err))
		return
	}
	msgq.Q.Send(SrvRoleName(pb.Server_Gateway), GateIDFromSesID(sesID), msgID, data, 0, sesID)
}

func SendToGate(serID int32, msg proto.Message, sesID uint64) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zap.L().Warn("send to gate error", zap.Error(err))
		return
	}
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Warn("send to gate error", zap.Error(err))
		return
	}
	msgq.Q.Send(SrvName(pb.Server_Gateway), serID, msgID, data, 0, sesID)
}
