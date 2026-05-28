package gnet

import (
	pb "server/api/pb"
	msgid2 "server/api/pb/msgid"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"server/pkg/idgen"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func GateIDFromSesID(gateID uint64) uint8 {
	return uint8(idgen.MachineID(int64(gateID)))
}

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
		zap.L().Warn("send to role error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid2.MsgIDS2C_name[int32(msgID)]),
			zap.Any("msg", msg),
			zap.Any("to", pb.Server_Gateway),
			zap.Uint8("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID))
		return
	}
	if trace.Rule.ShouldLog(msgID, roleID, sesID) {
		zap.L().Info(">>> msg.send: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid2.MsgIDS2C_name[int32(msgID)]),
			zap.Any("msg", msg),
			zap.Any("to", pb.Server_Gateway),
			zap.Uint8("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID),
		)
	}
}

func SendToSrv(serType pb.Server, serID uint8, msg proto.Message, actorID uint64, sesID uint64) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.Any("serName", serType), zap.Uint8("serID", serID))
		return
	}
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.Any("serName", serType), zap.Uint8("serID", serID))
		return
	}
	err = msgq.Q.Send(serType, serID, msgID, data, actorID, sesID)
	if err != nil {
		zap.L().Warn(">>> send to srv error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid2.MsgIDS2S_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.Any("to", serType),
			zap.Uint8("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID))
		return
	}
	if trace.Rule.ShouldLog(msgID, actorID, sesID) {
		zap.L().Info(">>> msg.send: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid2.MsgIDS2S_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.Any("to", serType),
			zap.Uint8("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID),
		)
	}
}

func SendToSrvAll(serType pb.Server, msg proto.Message, actorID uint64, sesID uint64) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.Any("serName", serType))
		return
	}
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Warn("send msg error", zap.Error(err), zap.Any("serName", serType))
		return
	}
	err = msgq.Q.SendAll(serType, msgID, data, actorID, sesID)
	if err != nil {
		zap.L().Warn(">>> send to all error: ",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid2.MsgIDS2S_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.Any("to", serType),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID))
		return
	}
	if trace.Rule.ShouldLog(msgID, actorID, sesID) {
		zap.L().Info(">>> msg.sendAll: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid2.MsgIDS2S_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.Any("to", serType),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID),
		)
	}
}

func SendToGate(msg proto.Message, sesID uint64) {
	SendToSrv(pb.Server_Gateway, GateIDFromSesID(sesID), msg, 0, sesID)
}

func SendToGame(serID uint8, msg proto.Message, sesID uint64, actorID uint64) {
	SendToSrv(pb.Server_Game, serID, msg, actorID, sesID)
}

func SendToAccount(msg proto.Message) {
	SendToSrv(pb.Server_Account, 0, msg, 0, 0)
}

func SendToCenter(msg proto.Message, actorID pb.ActorID) {
	SendToSrv(pb.Server_Center, 0, msg, uint64(actorID), 0)
}
