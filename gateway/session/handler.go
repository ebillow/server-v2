package session

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/gnet/trace"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

var (
	cliRouter = newCRouter()
	serRouter = newSRouter()
)

func C() *CRouter {
	return cliRouter
}
func S() *SRouter {
	return serRouter
}

type CRouter struct {
	*router.MsgRouter
}

func newCRouter() *CRouter {
	return &CRouter{MsgRouter: router.NewMsgRouter()}
}

// Msg 注册客户端发来的消息处理函数
func (rt *CRouter) Msg(msgID msgid.MsgIDC2S, df func(msg proto.Message, s *Session)) {
	err := rt.Register(uint32(msgID), pb.NewFuncC2S(msgID), func(msg proto.Message, c gctx.Context) {
		df(msg, c.U.(*Session))
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

func (rt *CRouter) Handle(msgID uint32, msgData []byte, s *Session) {
	node, err := rt.GetHandler(msgID)
	if err != nil {
		zap.L().Error("api not exist",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
		return
	}

	msgPB, err := rt.ParseMsg(node, msgData)
	if err != nil {
		zap.L().Error("parse msg error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
		return
	}

	if trace.Rule.ShouldLog(msgID, 0, s.Id) {
		info := "<<< msg.recv:" + msgid.MsgIDC2S_name[int32(msgID)]
		zap.L().Info(info,
			zap.Uint32("msgID", msgID),
			zap.Inline(s),
			zap.Any("data", msgPB),
			logger.Blue.Field(),
		)
	}

	node.HandleFunc(msgPB, gctx.Context{U: s})
}

type SRouter struct {
	*router.MsgRouter
}

func newSRouter() *SRouter {
	return &SRouter{MsgRouter: router.NewMsgRouter()}
}

// Msg 注册客户端发来的消息处理函数
func (rt *SRouter) Msg(msgID msgid.MsgIDS2S, df func(msg proto.Message, s *Session)) {
	err := rt.Register(uint32(msgID), pb.NewFuncS2S(msgID), func(msg proto.Message, c gctx.Context) {
		df(msg, c.U.(*Session))
	})
	if err != nil {
		zap.L().Error("Register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msgID)]))
	}
}

func (rt *SRouter) Handle(msg *pb.NatsMsg, raw *nats.Msg, s *Session) {
	err := rt.HandleMsg(gctx.Context{Msg: msg, Raw: raw, U: s, MsgName: msgid.MsgIDS2S_name})
	if err != nil {
		zap.L().Warn("hand msg failed",
			zap.Uint32("msgID", msg.MsgID),
			zap.String("from", flag.SrvName(msg.SerType)),
			zap.Int32("idx", msg.SerID),
			zap.Uint64("sessID", msg.SesID),
			zap.Uint64("roleID", msg.RoleID),
			zap.String("msg name", msgid.MsgIDS2S_name[int32(msg.MsgID)]))

		return
	}
}
