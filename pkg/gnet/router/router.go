package router

import (
	"reflect"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/game/role"
	"server/pkg/gnet/gctx"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var (
	_ role.IRouter = &MsgRouter{}
)

var (
	msgRouter = NewMsgRouter(int32(msgid.MsgIDMax_C2SMax))
)

func R() *MsgRouter {
	return msgRouter
}

func On[T pb.VTMessage](df func(c gctx.Context, req T)) {
	register(false, df)
}

// OnP 消息处理，使用对象池，req不能传递到其它协程，不能持有
func OnP[T pb.VTMessage](df func(c gctx.Context, req T)) {
	register(true, df)
}

func register[T pb.VTMessage](usePool bool, df func(c gctx.Context, req T)) {
	var zero T
	msgType := reflect.TypeOf(zero)
	msgID, ok := pb.GetMsgIDS2S(zero)
	if !ok {
		msgID, ok = pb.GetMsgIDC2S(zero)
	}
	if !ok {
		zap.L().Fatal("Register failed: message type not found in TypeMeta",
			zap.String("type", msgType.String()))
	}

	// 提取出指针底层的结构体
	elemType := msgType.Elem()

	createFunc := func() pb.VTMessage {
		return reflect.New(elemType).Interface().(pb.VTMessage)
	}

	handleFunc := func(c gctx.Context, msg proto.Message) {
		df(c, msg.(T))
	}

	err := R().Register(msgID, createFunc, usePool, handleFunc)
	if err != nil {
		zap.L().Fatal("Register failed: duplicate register",
			zap.String("msgType", msgType.String()),
			zap.Uint32("msgID", msgID),
			zap.Error(err))
	}
}
