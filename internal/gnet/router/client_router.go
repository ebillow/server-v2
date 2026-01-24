package snet

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/internal/pb"
	"server/internal/pb/msgid"
	"server/internal/util"
)

type ClientRouter struct {
	*roleMsgRouter
}

func newClientRouter(size uint32) *ClientRouter {
	return &ClientRouter{roleMsgRouter: newRoleMsgRouter(size)}
}

// Register 注册客户端发来的消息处理函数
func (rt *ClientRouter) Register(msgID msgid.MsgIDC2S, df func(msg proto.Message, r *role.Role)) {
	if !netStart.Load() {
		zap.L().Error("register msg handle failed, you mast Register before Serve or Connect",
			zap.Any("msgID", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]),
			zap.Stack("stack"))
		return
	}
	err := rt.register(uint32(msgID), pb.NewFuncC2S(msgID), df)
	if err != nil {
		zap.L().Error("register error",
			zap.Error(err),
			zap.Any("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
	}
}

func (rt *ClientRouter) Handle(msg *nats.Msg, r *role.Role) error {
	msgID := util.ParseUint32(msg.Header.Get("msg_id"))
	err := rt.handle(msgID, msg.Data, r)
	if err != nil {
		zap.L().Warn("hand msg failed",
			zap.Uint32("msg id", msgID),
			zap.String("msg name", msgid.MsgIDC2S_name[int32(msgID)]))
		return err
	}

	return nil
}
