package gnet

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/flag"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"server/pkg/logger"
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
	serName := flag.SrvName(pb.Server_Gateway)
	serID := GateIDFromSesID(sesID)
	err = msgq.Q.ForwardToRole(serName, serID, msgID, data, roleID, sesID)
	if err != nil {
		zap.L().Warn("send to role error", zap.Error(err))
		return
	}

	if trace.Rule.ShouldLog(msgID, roleID, sesID) {
		zap.L().Info(">>> msg.send: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2C_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.String("to", serName),
			zap.Int32("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID),
			logger.Magenta.Field(),
		)
	}
}

func SendToSrv(serType pb.Server, serID int32, msg proto.Message, roleID uint64, sesID uint64) {
	serName := flag.SrvName(serType)
	data, err := proto.Marshal(msg)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.String("serName", serName), zap.Int32("serID", serID))
		return
	}
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.String("serName", serName), zap.Int32("serID", serID))
		return
	}
	err = msgq.Q.Send(serName, serID, msgID, data, roleID, sesID)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.String("serName", serName), zap.Int32("serID", serID))
		return
	}
	if trace.Rule.ShouldLog(msgID, roleID, sesID) {
		zap.L().Info(">>> msg.send: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2C_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.String("to", serName),
			zap.Int32("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID),
			logger.Magenta.Field(),
		)
	}
}

func SendToGate(msg proto.Message, sesID uint64) {
	SendToSrv(pb.Server_Gateway, GateIDFromSesID(sesID), msg, 0, sesID)
}

func SendToGame(serID int32, msg proto.Message, sesID uint64, roleID uint64) {
	SendToSrv(pb.Server_Game, serID, msg, roleID, sesID)
}

func SendToAccount(serID int32, msg proto.Message, sesID uint64) {
	SendToSrv(pb.Server_Account, serID, msg, 0, sesID)
}
