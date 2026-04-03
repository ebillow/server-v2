package gnet

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

func SendToRole(msg proto.Message, sesID uint64, roleID uint64) {
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
	serID := GateIDFromSesID(sesID)
	err = msgq.Q.ForwardToRole(pb.Server_Gateway, serID, msgID, data, roleID, sesID)
	if err != nil {
		zap.L().Warn("send to role error", zap.Error(err))
		return
	}

	if trace.Rule.ShouldLog(msgID, roleID, sesID) {
		zap.L().Info(">>> msg.send: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2C_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.Any("to", pb.Server_Gateway),
			zap.Int32("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID),
		)
	}
}

func SendToSrv(serType pb.Server, serID int32, msg proto.Message, roleID uint64, sesID uint64) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.Any("serName", serType), zap.Int32("serID", serID))
		return
	}
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.Any("serName", serType), zap.Int32("serID", serID))
		return
	}
	err = msgq.Q.Send(serType, serID, msgID, data, roleID, sesID)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.Any("serName", serType), zap.Int32("serID", serID))
		return
	}
	if trace.Rule.ShouldLog(msgID, roleID, sesID) {
		zap.L().Info(">>> msg.send: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.Any("to", serType),
			zap.Int32("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID),
		)
	}
}

func SendToGate(msg proto.Message, sesID uint64) {
	SendToSrv(pb.Server_Gateway, GateIDFromSesID(sesID), msg, 0, sesID)
}

func SendToGame(serID int32, msg proto.Message, sesID uint64, roleID uint64) {
	SendToSrv(pb.Server_Game, serID, msg, roleID, sesID)
}

func SendToAccount(msg proto.Message) {
	SendToSrv(pb.Server_Account, 0, msg, 0, 0)
}
