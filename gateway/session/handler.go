package session

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/gnet/router"
	"server/pkg/gnet/trace"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

var cliMsgHandler = newRouter(MaxMsgCount)

func CRouter() *Router {
	return cliMsgHandler
}

type Router struct {
	*router.MsgRouter
}

func newRouter(size uint32) *Router {
	return &Router{MsgRouter: router.NewMsgRouter(size)}
}

// RoleMsg 注册客户端发来的消息处理函数
func (rt *Router) RoleMsg(msgID msgid.MsgIDC2S, df func(msg proto.Message, s *Session)) {
	err := rt.Register(uint32(msgID), pb.NewFuncC2S(msgID), func(msg proto.Message, c router.Context) {
		df(msg, c.U.(*Session))
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

func (rt *Router) Handle(msgID uint32, msgData []byte, s *Session) error {
	msgPB, err := rt.HandleMsg(msgID, msgData, router.Context{U: s})
	if err != nil {
		zap.L().Warn("hand msg failed",
			zap.Uint32("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
		return err
	}

	if trace.Rule.ShouldLog(msgID, 0, s.Id) {
		info := "<<< handle: " + msgid.MsgIDC2S_name[int32(msgID)]
		// bytes, _ := json.Marshal(pkt.IMsg)
		zap.L().Info(info,
			zap.Uint32("msgID", msgID),

			zap.Uint64("sessID", s.Id),
			zap.Any("data", msgPB),
			logger.Blue.Field(),
		)
	}
	return nil
}

//
// func (s *Session) onRecvGameMsg(msgs *nats.Msg) {
// 	for _, msg := range msgs.Msgs {
// 		if msg.ID > uint32(msgid.MsgIDS2C_S2CGateHandle) {
// 			s.SendBytes(msg.ID, msg.Msg)
// 			if IsTraceProto() && msg.ID != uint32(msgid.MsgIDS2C_S2CHeartBeat) {
// 				logger.Infof("%s forward to cli [%d]%s", s.String(), msg.ID, msgid.MsgIDS2C_name[int32(msg.ID)])
// 			}
// 		} else {
// 			if err := serMsgHandler.Handle(msg.ID, msg.Msg, s); err != nil {
// 				logger.Warnf("%s recv invalid msg from game [%d]", s.String(), msg.ID)
// 			}
// 		}
// 	}
// }
