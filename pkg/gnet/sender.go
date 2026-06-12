package gnet

import (
	"fmt"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/pool"
	"server/pkg/gnet/trace"
	"server/pkg/idgen"
	"server/pkg/util"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

func GateIDFromSesID(gateID uint64) uint8 {
	return uint8(idgen.MachineID(int64(gateID)))
}

// SendToClient 发送给前端
func SendToClient[T pb.VTMessage](msgID uint32, msg T, sesID uint64, roleID uint64) {
	pBuf, buf := ensureBuf(msg.SizeVT())

	n, err := msg.MarshalToSizedBufferVT(buf)
	if err != nil {
		pool.BufPool512.Put(pBuf)
		zap.L().Warn("send marshal vt error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
		)
		return
	}

	buf = buf[:n]

	serID := GateIDFromSesID(sesID)
	err = msgq.Q.ForwardToClient(serID, msgID, buf, roleID, sesID)

	pool.BufPool512.Put(pBuf)

	if err != nil {
		zap.L().Warn("send to role error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2C_name[int32(msgID)]),
			zap.String("to", pb.Server_name[int32(pb.Server_Gateway)]),
			zap.Uint8("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID))
		return
	}
	if trace.Rule.ShouldLog(msgID, roleID, sesID) {
		str, _ := sonic.MarshalString(msg)
		zap.L().Info("send",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2C_name[int32(msgID)]),
			zap.String("msg", str),
			zap.String("to", pb.Server_name[int32(pb.Server_Gateway)]),
			zap.Uint8("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID),
		)
	}
}

// SendToSrv 发送给指定服务
func SendToSrv[T pb.VTMessage](
	serType pb.Server,
	serID uint8,
	msgID uint32,
	msg T,
	actorID uint64,
	sesID uint64,
) {
	pBuf, buf := ensureBuf(msg.SizeVT())

	_, err := msg.MarshalToSizedBufferVT(buf)
	if err != nil {
		pool.BufPool512.Put(pBuf)
		zap.L().Warn("send marshal vt error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.Uint8("serID", serID),
		)
		return
	}

	err = msgq.Q.SendTo(serType, serID, msgID, buf, actorID, sesID)

	pool.BufPool512.Put(pBuf)

	if err != nil {
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
		str, _ := sonic.MarshalString(msg)
		zap.L().Info("send",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.String("msg", str),
			zap.String("to", pb.Server_name[int32(serType)]),
			zap.Uint8("idx", serID),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID),
		)
	}
}

// SendToAny 组发送. 随机一个能收到
func SendToAny[T pb.VTMessage](
	serType pb.Server,
	msgID uint32,
	msg T,
	actorID uint64,
	sesID uint64,
) {
	pBuf, buf := ensureBuf(msg.SizeVT())

	_, err := msg.MarshalToSizedBufferVT(buf)
	if err != nil {
		pool.BufPool512.Put(pBuf)
		zap.L().Warn("send marshal vt error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
		)
		return
	}
	err = msgq.Q.SendToAny(serType, msgID, buf, actorID, sesID)

	pool.BufPool512.Put(pBuf)

	if err != nil {
		str, _ := sonic.MarshalString(msg)
		zap.L().Warn(">>> send to all error: ",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.String("msg", str),
			zap.String("to", pb.Server_name[int32(serType)]),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID))
		return
	}
	if trace.Rule.ShouldLog(msgID, actorID, sesID) {
		str, _ := sonic.MarshalString(msg)
		zap.L().Info(">>> msg.sendAll: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.String("msg", str),
			zap.String("to", pb.Server_name[int32(serType)]),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID),
		)
	}
}

// SendToGroup 同类型所有的服节点都能收到
func SendToGroup[T pb.VTMessage](
	serType pb.Server,
	msgID uint32,
	msg T,
	actorID uint64,
	sesID uint64,
) {
	pBuf, buf := ensureBuf(msg.SizeVT())

	_, err := msg.MarshalToSizedBufferVT(buf)
	if err != nil {
		pool.BufPool512.Put(pBuf)
		zap.L().Warn("send marshal vt error",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
		)
		return
	}
	err = msgq.Q.SendToGroup(serType, msgID, buf, actorID, sesID)

	pool.BufPool512.Put(pBuf)

	if err != nil {
		str, _ := sonic.MarshalString(msg)
		zap.L().Warn(">>> send to all error: ",
			zap.Error(err),
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.String("to", pb.Server_name[int32(serType)]),
			zap.String("msg", str),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID))
		return
	}
	if trace.Rule.ShouldLog(msgID, actorID, sesID) {
		str, _ := sonic.MarshalString(msg)
		zap.L().Info(">>> msg.sendAll: ",
			zap.Uint32("msgID", msgID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.String("msg", str),
			zap.String("to", pb.Server_name[int32(serType)]),
			zap.Uint64("sessID", sesID),
			zap.Uint64("actorID", actorID),
		)
	}
}

func ensureBuf(size int) (*[]byte, []byte) {
	pBuf := pool.BufPool512.Get()
	if cap(*pBuf) < size {
		*pBuf = make([]byte, size)
	}
	return pBuf, (*pBuf)[:size]
}

func SendToGate[T pb.VTMessage](msgID msgid.MsgIDS2S, msg T, sesID uint64) {
	if util.Debug {
		realMsgID, ok := pb.GetMsgIDS2S(msg)
		if !ok {
			zap.L().Error("[SendT] GetMsgIDS2C error")
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2C_name[int32(msgID)]),
			)
			return
		}
	}
	SendToSrv(pb.Server_Gateway, GateIDFromSesID(sesID), uint32(msgID), msg, 0, sesID)
}

func SendToGame[T pb.VTMessage](serID uint8, msgID msgid.MsgIDS2S, msg T, sesID uint64, actorID uint64) {
	if util.Debug {
		realMsgID, ok := pb.GetMsgIDS2S(msg)
		if !ok {
			zap.L().Error("[SendT] GetMsgIDS2C error")
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2S_name[int32(msgID)]),
			)
			return
		}
	}
	SendToSrv(pb.Server_Game, serID, uint32(msgID), msg, actorID, sesID)
}

func SendToAccount[T pb.VTMessage](msgID msgid.MsgIDS2S, msg T) {
	if util.Debug {
		realMsgID, ok := pb.GetMsgIDS2S(msg)
		if !ok {
			zap.L().Error("[SendT] GetMsgIDS2C error")
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2S_name[int32(msgID)]),
			)
			return
		}
	}
	SendToAny(pb.Server_Account, uint32(msgID), msg, 0, 0)
}

func SendToCenter[T pb.VTMessage](msgID msgid.MsgIDS2S, msg T, actorID pb.ActorID) {
	if util.Debug {
		realMsgID, ok := pb.GetMsgIDS2S(msg)
		if !ok {
			zap.L().Error("[SendT] GetMsgIDS2C error")
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2S_name[int32(msgID)]),
			)
			return
		}
	}
	SendToAny(pb.Server_Center, uint32(msgID), msg, uint64(actorID), 0)
}

// -------------------------非热点路径可以使用----------------------

func SendToGateS[T pb.VTMessage](msg T, sesID uint64) {
	msgID, ok := pb.GetMsgIDS2S(msg)
	if !ok {
		zap.L().Error("send msg error, msg id not exists", zap.String("type", fmt.Sprintf("%T", msg)))
		return
	}
	SendToSrv(pb.Server_Gateway, GateIDFromSesID(sesID), msgID, msg, 0, sesID)
}

func SendToGameS[T pb.VTMessage](serID uint8, msg T, sesID uint64, actorID uint64) {
	msgID, ok := pb.GetMsgIDS2S(msg)
	if !ok {
		zap.L().Error("send msg error, msg id not exists", zap.String("type", fmt.Sprintf("%T", msg)))
		return
	}
	SendToSrv(pb.Server_Game, serID, msgID, msg, actorID, sesID)
}

func SendToAccountS[T pb.VTMessage](msg T) {
	msgID, ok := pb.GetMsgIDS2S(msg)
	if !ok {
		zap.L().Error("send msg error, msg id not exists", zap.String("type", fmt.Sprintf("%T", msg)))
		return
	}
	SendToAny(pb.Server_Account, msgID, msg, 0, 0)
}

func SendToCenterS[T pb.VTMessage](msg T, actorID pb.ActorID) {
	msgID, ok := pb.GetMsgIDS2S(msg)
	if !ok {
		zap.L().Error("send msg error, msg id not exists", zap.String("type", fmt.Sprintf("%T", msg)))
		return
	}
	SendToAny(pb.Server_Center, msgID, msg, uint64(actorID), 0)
}
