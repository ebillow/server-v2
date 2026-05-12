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

type ServerRouter struct {
	*MsgRouter
}

func newServerRouter(max int32) *ServerRouter {
	return &ServerRouter{MsgRouter: NewMsgRouter(max)}
}

// On 注册服务器间消息，并且是发给指定角色的消息处理函数
func (rt *ServerRouter) On(msgID msgid.MsgIDS2S, df func(c gctx.Context, msg proto.Message, r *role.Role)) {
	if flag.IsReady() {
		zap.L().Error("注册消息失败，必须在监听前注册",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}
	err := rt.register(uint32(msgID), pb.NewFuncS2S(msgID), func(c gctx.Context, msg proto.Message) {
		df(c, msg, c.U.(*role.Role))
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
	}
}

// OnG 注册服务器间消息，不是角色消息处理函数
func (rt *ServerRouter) OnG(msgID msgid.MsgIDS2S, df func(c gctx.Context, msg proto.Message)) {
	if flag.IsReady() {
		zap.L().Error("注册消息失败，必须在监听前注册",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}
	err := rt.register(uint32(msgID), pb.NewFuncS2S(msgID), func(c gctx.Context, msg proto.Message) {
		df(c, msg)
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
	}
}

func (rt *ServerRouter) Handle(ctx gctx.Context) {
	err := rt.handleMsg(ctx, func(msgPB proto.Message) {
		if trace.Rule.ShouldLog(ctx.MsgID, ctx.RoleID, ctx.SesID) {
			zap.L().Info("<<< msg.recv:",
				zap.String("msgName", msgid.MsgIDS2S_name[int32(ctx.MsgID)]),
				zap.Inline(&ctx),
			)
		}
	})
	if err != nil {
		zap.L().Warn("HandleWithRole failed", zap.Inline(&ctx), zap.Error(err))
		return
	}
}
