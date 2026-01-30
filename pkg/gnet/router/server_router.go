package router

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

type ServerRouter struct {
	*MsgRouter
}

func newServerRouter(size uint32) *ServerRouter {
	return &ServerRouter{MsgRouter: NewMsgRouter(size)}
}

// RoleMsg 注册服务器间消息，并且是发给指定角色的消息处理函数
func (rt *ServerRouter) RoleMsg(msgID msgid.MsgIDS2S, df func(msg proto.Message, r *role.Role, qm *nats.Msg)) {
	if netStart.Load() {
		zap.L().Error("Register msg Handle failed, you mast RoleMsg before Serve or Connect",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}
	err := rt.Register(uint32(msgID), pb.NewFuncS2S(msgID), func(msg proto.Message, c Context) {
		df(msg, c.U.(*role.Role), c.Msg)
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
	}
}

func (rt *ServerRouter) HandleWithRole(msg *nats.Msg, r *role.Role) {
	msgID := msgq.MsgID(msg)
	msgPB, err := rt.HandleMsg(msgID, msg.Data, Context{Msg: msg, U: r})
	if err != nil {
		zap.L().Warn("hand msg failed",
			zap.Uint32("msg id", msgID),
			zap.Any("head", msg.Header),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
		return
	}

	if trace.Rule.ShouldLog(msgID, r.ID, r.SesID) {
		info := "<<< msg.recv: " + msgid.MsgIDS2S_name[int32(msgID)]
		// bytes, _ := json.Marshal(pkt.IMsg)
		zap.L().Info(info,
			zap.Uint32("msgID", msgID),
			zap.String("from", msgq.ServerName(msg)),
			zap.Int32("idx", msgq.ServerID(msg)),
			zap.Uint64("sessID", r.SesID),
			zap.Uint64("roleID", r.ID),
			zap.Any("data", msgPB),
			logger.Blue.Field(),
		)
	}
}

// Msg 注册服务器间消息，不是角色消息处理函数
func (rt *ServerRouter) Msg(msgID msgid.MsgIDS2S, df func(msg proto.Message, qm *nats.Msg)) {
	if netStart.Load() {
		zap.L().Error("Register msg Handle failed, you mast RoleMsg before Serve or Connect",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}
	err := rt.Register(uint32(msgID), pb.NewFuncS2S(msgID), func(msg proto.Message, c Context) {
		df(msg, c.Msg)
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
	}
}

func (rt *ServerRouter) Handle(msg *nats.Msg) {
	msgID := msgq.MsgID(msg)
	msgPB, err := rt.HandleMsg(msgID, msg.Data, Context{Msg: msg})
	if err != nil {
		zap.L().Warn("hand msg failed",
			zap.Uint32("msg id", msgID),
			zap.Any("head", msg.Header),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
		return
	}

	roleID := msgq.RoleID(msg)
	sesID := msgq.SessionID(msg)
	if trace.Rule.ShouldLog(msgID, roleID, sesID) {
		info := "<<< msg.recv: " + msgid.MsgIDS2S_name[int32(msgID)]
		// bytes, _ := json.Marshal(pkt.IMsg)
		zap.L().Info(info,
			zap.Uint32("msgID", msgID),
			zap.String("from", msgq.ServerName(msg)),
			zap.Int32("idx", msgq.ServerID(msg)),
			zap.Uint64("sessID", sesID),
			zap.Uint64("roleID", roleID),
			zap.Any("data", msgPB),
			logger.Blue.Field(),
		)
	}

	return
}
