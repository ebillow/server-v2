package router

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/trace"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

type ClientRouter struct {
	*MsgRouter
}

func newClientRouter() *ClientRouter {
	return &ClientRouter{MsgRouter: NewMsgRouter()}
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

	err := rt.Register(uint32(msgID), pb.NewFuncC2S(msgID), func(c gctx.Context, msg proto.Message) {
		df(c, msg, c.U.(*role.Role))
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

func (rt *ClientRouter) Handle(natMsg *pb.NatsMsg, raw *nats.Msg, r *role.Role) {
	err := rt.HandleMsg(gctx.Context{Msg: natMsg, Raw: raw, U: r}, func(msgPB proto.Message) {
		if trace.Rule.ShouldLog(natMsg.MsgID, natMsg.RoleID, natMsg.SesID) {
			zap.L().Info("<<< msg.recv:",
				zap.Uint32("msgID", natMsg.MsgID),
				zap.String("msgName", msgid.MsgIDC2S_name[int32(natMsg.MsgID)]),
				zap.String("from", flag.SrvName(natMsg.SerType)),
				zap.Int32("serID", natMsg.SerID),
				zap.Any("data", msgPB),
				zap.Uint64("roleID", natMsg.RoleID),
				zap.Uint64("sesID", natMsg.SesID),
			)
		}
	})
	if err != nil {
		zap.L().Warn("HandleMsg failed",
			zap.Uint32("msgID", natMsg.MsgID),
			zap.String("from", flag.SrvName(natMsg.SerType)),
			zap.Int32("idx", natMsg.SerID),
			zap.Uint64("sessID", natMsg.SesID),
			zap.Uint64("roleID", natMsg.RoleID),
			zap.Error(err),
			zap.String("msgName", msgid.MsgIDC2S_name[int32(natMsg.MsgID)]))
		return
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

	err := rt.Register(uint32(msgID), pb.NewFuncC2S(msgID), func(c gctx.Context, msg proto.Message) {
		df(c, msg)
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

func (rt *ClientRouter) HandleG(natMsg *pb.NatsMsg, raw *nats.Msg) {
	err := rt.HandleMsg(gctx.Context{Msg: natMsg, Raw: raw}, func(msgPB proto.Message) {
		if trace.Rule.ShouldLog(natMsg.MsgID, natMsg.RoleID, natMsg.SesID) {
			zap.L().Info("<<< msg.recv:",
				zap.Uint32("msgID", natMsg.MsgID),
				zap.String("msgName", msgid.MsgIDC2S_name[int32(natMsg.MsgID)]),
				zap.String("from", flag.SrvName(natMsg.SerType)),
				zap.Int32("serID", natMsg.SerID),
				zap.Any("data", msgPB),
				zap.Uint64("roleID", natMsg.RoleID),
				zap.Uint64("sesID", natMsg.SesID),
			)
		}
	})
	if err != nil {
		zap.L().Warn("HandleMsg failed",
			zap.Uint32("msgID", natMsg.MsgID),
			zap.String("from", flag.SrvName(natMsg.SerType)),
			zap.Int32("idx", natMsg.SerID),
			zap.Uint64("sessID", natMsg.SesID),
			zap.Uint64("roleID", natMsg.RoleID),
			zap.Error(err),
			zap.String("msgName", msgid.MsgIDC2S_name[int32(natMsg.MsgID)]))
		return
	}
}
