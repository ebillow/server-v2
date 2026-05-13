package router

import (
	"server/game/role"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/trace"
	"server/pkg/pb"
	"server/pkg/pb/msgid"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type ClientRouter struct {
	*MsgRouter
}

func newClientRouter(max int32) *ClientRouter {
	return &ClientRouter{MsgRouter: NewMsgRouter(max)}
}

// On 注册客户端发来的消息处理函数，处理函数带role
func (rt *ClientRouter) On(msgID msgid.MsgIDC2S, df func(c gctx.Context, msg proto.Message, r *role.Role)) {
	if flag.IsReady() {
		zap.L().Error("注册消息失败，必须在监听前注册",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}

	err := rt.register(uint32(msgID), pb.NewFuncC2S(msgID), func(c gctx.Context, msg proto.Message) {
		df(c, msg, c.U.(*role.Role))
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

// OnG 注册客户端发来的消息处理函数,不带role
func (rt *ClientRouter) OnG(msgID msgid.MsgIDC2S, df func(c gctx.Context, msg proto.Message)) {
	if flag.IsReady() {
		zap.L().Error("注册消息失败，必须在监听前注册",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}

	err := rt.register(uint32(msgID), pb.NewFuncC2S(msgID), func(c gctx.Context, msg proto.Message) {
		df(c, msg)
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

func (rt *ClientRouter) Handle(ctx gctx.Context) {
	err := rt.handleMsg(ctx, func(msgPB proto.Message) {
		if trace.Rule.ShouldLog(ctx.MsgID, ctx.ActorID, ctx.SesID) {
			zap.L().Info("<<< msg.recv:",
				zap.String("msgName", msgid.MsgIDC2S_name[int32(ctx.MsgID)]),
				zap.Any("data", msgPB),
				zap.Inline(&ctx),
			)
		}
	})
	if err != nil {
		zap.L().Warn("HandleMsg failed", zap.Inline(&ctx), zap.Error(err))
		return
	}
}
