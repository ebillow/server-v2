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

type ServerRouter struct {
	*MsgRouter
}

func newServerRouter() *ServerRouter {
	return &ServerRouter{MsgRouter: NewMsgRouter()}
}

// On 注册服务器间消息，并且是发给指定角色的消息处理函数
func (rt *ServerRouter) On(msgID msgid.MsgIDS2S, df func(c gctx.Context, msg proto.Message, r *role.Role)) {
	if netStart.Load() {
		zap.L().Error("注册消息失败，必须在监听前注册",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}
	err := rt.Register(uint32(msgID), pb.NewFuncS2S(msgID), func(c gctx.Context, msg proto.Message) {
		df(c, msg, c.U.(*role.Role))
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
	}
}

func (rt *ServerRouter) Handle(natMsg *pb.NatsMsg, raw *nats.Msg, r *role.Role) {
	err := rt.HandleMsg(gctx.Context{Msg: natMsg, Raw: raw, U: r, MsgName: msgid.MsgIDS2S_name})
	if err != nil {
		zap.L().Warn("HandleWithRole failed",
			zap.Uint32("msgID", natMsg.MsgID),
			zap.String("from", flag.SrvName(natMsg.SerType)),
			zap.Int32("idx", natMsg.SerID),
			zap.Uint64("roleID", natMsg.RoleID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(natMsg.MsgID)]))
		return
	}
}

// OnG 注册服务器间消息，不是角色消息处理函数
func (rt *ServerRouter) OnG(msgID msgid.MsgIDS2S, df func(c gctx.Context, msg proto.Message)) {
	if netStart.Load() {
		zap.L().Error("注册消息失败，必须在监听前注册",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}
	err := rt.Register(uint32(msgID), pb.NewFuncS2S(msgID), func(c gctx.Context, msg proto.Message) {
		df(c, msg)
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
	}
}

func (rt *ServerRouter) HandleG(natMsg *pb.NatsMsg, raw *nats.Msg) {
	err := rt.HandleMsg(gctx.Context{Msg: natMsg, Raw: raw, MsgName: msgid.MsgIDS2S_name})
	if err != nil {
		zap.L().Warn("Handle failed",
			zap.Uint32("msgID", natMsg.MsgID),
			zap.String("from", flag.SrvName(natMsg.SerType)),
			zap.Int32("idx", natMsg.SerID),
			zap.String("msgName", msgid.MsgIDS2S_name[int32(natMsg.MsgID)]))
		return
	}
}
