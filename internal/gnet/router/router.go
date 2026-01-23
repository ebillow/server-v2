package snet

import (
	"errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/game/role"
	"server/internal/pb"
	"server/internal/pb/msgid"
	"sync/atomic"
)

var (
	errAPINotFind = errors.New("api not find")
)

var (
	cliMsgHandler     = newRoleMsgRouter(65536)
	serRoleMsgHandler = newRoleMsgRouter(65536)
	netStart          atomic.Bool
)

// RegisterCli 注册客户端发来的消息处理函数
func RegisterCli(msgID msgid.MsgIDC2S, df func(msg proto.Message, r *role.Role)) {
	if !netStart.Load() {
		zap.L().Error("register msg handle failed, you mast register before Serve or Connect",
			zap.Any("msgID", msgID), zap.Stack("stack"))
		return
	}
	cliMsgHandler.register(uint16(msgID), pb.NewFuncC2S(msgID), df)
}

// Register 注册服务器发来的消息处理函数
func Register(msgID msgid.MsgIDS2S, df func(msg proto.Message, r *role.Role)) {
	if !netStart.Load() {
		zap.L().Error("register msg handle failed, you mast register before Serve or Connect",
			zap.Any("msgID", msgID), zap.Stack("stack"))
		return
	}
	serRoleMsgHandler.register(uint16(msgID), pb.NewFuncS2S(msgID), df)
}
