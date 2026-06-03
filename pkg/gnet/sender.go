package gnet

import (
	"fmt"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"server/pkg/idgen"
	"server/pkg/util"

	"go.uber.org/zap"
)

func GateIDFromSesID(gateID uint64) uint8 {
	return uint8(idgen.MachineID(int64(gateID)))
}

func SendToRole[T pb.VTMessage](msgID uint32, msg T, sesID uint64, roleID uint64) {
	pBuf := Get()
	size := msg.SizeVT()
	if cap(*pBuf) < size {
		*pBuf = make([]byte, size)
	}
	buf := (*pBuf)[:size]

	n, err := msg.MarshalToSizedBufferVT(buf)
	if err != nil {
		Put(pBuf)
		zap.L().Warn("send marshal vt error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
		)
		return
	}

	buf = buf[:n]

	serID := GateIDFromSesID(sesID)
	err = msgq.Q.ForwardToRole(pb.Server_Gateway, serID, msgID, *pBuf, roleID, sesID)

	Put(pBuf)

	if err != nil {
		zap.L().Warn("send to role error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2C_name[int32(msgID)]),
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
			zap.String("msgName", msgid.MsgIDS2C_name[int32(msgID)]),
			zap.Any("msg", msg),
			zap.Any("to", pb.Server_Gateway),
			zap.Uint8("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID),
		)
	}
}

func SendToSrv[T pb.VTMessage](
	serType pb.Server,
	serID uint8,
	msgID uint32,
	msg T,
	actorID uint64,
	sesID uint64,
) {
	pBuf := Get()

	size := msg.SizeVT()
	if cap(*pBuf) < size {
		*pBuf = make([]byte, size)
	}
	buf := (*pBuf)[:size]

	_, err := msg.MarshalToSizedBufferVT(buf)
	if err != nil {
		Put(pBuf)
		zap.L().Warn("send marshal vt error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.Uint8("serID", serID),
		)
		return
	}

	if err = msgq.Q.Send(serType, serID, msgID, buf, actorID, sesID); err != nil {
		Put(pBuf)
		zap.L().Warn("send to srv error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.Uint8("serID", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID),
		)
		return
	}

	if trace.Rule.ShouldLog(msgID, actorID, sesID) {
		zap.L().Info("msg.send",
			zap.Uint32("msgID", msgID),
			zap.Uint8("serID", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID),
		)
	}
	Put(pBuf)
}

func SendToSrvAll[T pb.VTMessage](
	serType pb.Server,
	msgID uint32,
	msg T,
	actorID uint64,
	sesID uint64,
) {
	pBuf := Get()
	var err error

	size := msg.SizeVT()
	if cap(*pBuf) < size {
		*pBuf = make([]byte, size)
	}
	buf := (*pBuf)[:size]

	_, err = msg.MarshalToSizedBufferVT(buf)
	if err != nil {
		Put(pBuf)
		zap.L().Warn("send marshal vt error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
		)
		return
	}
	err = msgq.Q.SendAll(serType, msgID, *pBuf, actorID, sesID)

	Put(pBuf)

	if err != nil {
		zap.L().Warn(">>> send to all error: ",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Any("data", msg), // todo 有反射
			zap.Any("to", serType),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID))
		return
	}
	if trace.Rule.ShouldLog(msgID, actorID, sesID) {
		zap.L().Info(">>> msg.sendAll: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Any("data", msg),
			zap.Any("to", serType),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID),
		)
	}
}

func SendToGate[T pb.VTMessage](msgID msgid.MsgIDS2S, msg T, sesID uint64) {
	if util.Debug {
		realMsgID, err := pb.GetMsgIDS2C(msg)
		if err != nil {
			zap.L().Error("[SendT] GetMsgIDS2C error",
				zap.Error(err),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2C_name[int32(msgID)]),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
	}
	SendToSrv(pb.Server_Gateway, GateIDFromSesID(sesID), uint32(msgID), msg, 0, sesID)
}

func SendToGame[T pb.VTMessage](serID uint8, msgID msgid.MsgIDS2S, msg T, sesID uint64, actorID uint64) {
	if util.Debug {
		realMsgID, err := pb.GetMsgIDS2S(msg)
		if err != nil {
			zap.L().Error("[SendT] GetMsgIDS2S error",
				zap.Error(err),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2S_name[int32(msgID)]),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
	}
	SendToSrv(pb.Server_Game, serID, uint32(msgID), msg, actorID, sesID)
}

func SendToAccount[T pb.VTMessage](msgID msgid.MsgIDS2S, msg T) {
	if util.Debug {
		realMsgID, err := pb.GetMsgIDS2S(msg)
		if err != nil {
			zap.L().Error("[SendT] GetMsgIDS2S error",
				zap.Error(err),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2S_name[int32(msgID)]),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
	}
	SendToSrv(pb.Server_Account, 0, uint32(msgID), msg, 0, 0)
}

func SendToCenter[T pb.VTMessage](msgID msgid.MsgIDS2S, msg T, actorID pb.ActorID) {
	if util.Debug {
		realMsgID, err := pb.GetMsgIDS2S(msg)
		if err != nil {
			zap.L().Error("[SendT] GetMsgIDS2S error",
				zap.Error(err),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2S_name[int32(msgID)]),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
	}
	SendToSrv(pb.Server_Center, 0, uint32(msgID), msg, uint64(actorID), 0)
}

func SendToGateS[T pb.VTMessage](msg T, sesID uint64) {
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Error("send msg error", zap.Error(err))
		return
	}
	SendToSrv(pb.Server_Gateway, GateIDFromSesID(sesID), msgID, msg, 0, sesID)
}

func SendToGameS[T pb.VTMessage](serID uint8, msg T, sesID uint64, actorID uint64) {
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Error("send msg error", zap.Error(err))
		return
	}
	SendToSrv(pb.Server_Game, serID, msgID, msg, actorID, sesID)
}

func SendToAccountS[T pb.VTMessage](msg T) {
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Error("send msg error", zap.Error(err))
		return
	}
	SendToSrv(pb.Server_Account, 0, msgID, msg, 0, 0)
}

func SendToCenterS[T pb.VTMessage](msg T, actorID pb.ActorID) {
	msgID, err := pb.GetMsgIDS2S(msg)
	if err != nil {
		zap.L().Error("send msg error", zap.Error(err))
		return
	}
	SendToSrv(pb.Server_Center, 0, msgID, msg, uint64(actorID), 0)
}
