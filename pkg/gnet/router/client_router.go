package router

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

type ClientRouter struct {
	*MsgRouter
}

func newClientRouter() *ClientRouter {
	return &ClientRouter{MsgRouter: NewMsgRouter()}
}

// RoleMsg 注册客户端发来的消息处理函数，处理函数带role
func (rt *ClientRouter) RoleMsg(msgID msgid.MsgIDC2S, df func(msg proto.Message, r *role.Role, c gctx.Context)) {
	if netStart.Load() {
		zap.L().Error("注册消息失败，必须在监听前注册",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}

	err := rt.Register(uint32(msgID), pb.NewFuncC2S(msgID), func(msg proto.Message, c gctx.Context) {
		df(msg, c.U.(*role.Role), c)
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

func (rt *ClientRouter) HandleWithRole(natMsg *pb.NatsMsg, raw *nats.Msg, r *role.Role) {
	err := rt.HandleMsg(gctx.Context{Msg: natMsg, Raw: raw, U: r, MsgName: msgid.MsgIDC2S_name})
	if err != nil {
		zap.L().Warn("HandleMsg failed",
			zap.Uint32("msgID", natMsg.MsgID),
			zap.String("from", flag.SrvName(natMsg.SerType)),
			zap.Int32("idx", natMsg.SerID),
			zap.Uint64("sessID", natMsg.SesID),
			zap.Uint64("roleID", natMsg.RoleID),
			zap.String("raw name", msgid.MsgIDC2S_name[int32(natMsg.MsgID)]))
		return
	}
}

// Msg 注册客户端发来的消息处理函数
func (rt *ClientRouter) Msg(msgID msgid.MsgIDC2S, df func(msg proto.Message, c gctx.Context)) {
	if netStart.Load() {
		zap.L().Error("注册消息失败，必须在监听前注册",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}

	err := rt.Register(uint32(msgID), pb.NewFuncC2S(msgID), func(msg proto.Message, c gctx.Context) {
		df(msg, c)
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

func (rt *ClientRouter) Handle(natMsg *pb.NatsMsg, raw *nats.Msg) {
	err := rt.HandleMsg(gctx.Context{Msg: natMsg, Raw: raw, MsgName: msgid.MsgIDC2S_name})
	if err != nil {
		zap.L().Warn("HandleMsg failed",
			zap.Uint32("msgID", natMsg.MsgID),
			zap.String("from", flag.SrvName(natMsg.SerType)),
			zap.Int32("idx", natMsg.SerID),
			zap.Uint64("sessID", natMsg.SesID),
			zap.Uint64("roleID", natMsg.RoleID),
			zap.String("raw name", msgid.MsgIDC2S_name[int32(natMsg.MsgID)]))
		return
	}
}
