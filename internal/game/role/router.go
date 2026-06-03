package role

import (
	"reflect"
	"server/api/pb"
	"server/pkg/gnet/gctx"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func On[T pb.VTMessage](df func(c gctx.Context, req T, r *Role)) {
	register(false, df)
}

func OnP[T pb.VTMessage](df func(c gctx.Context, req T, r *Role)) {
	register(true, df)
}

func register[T pb.VTMessage](usePool bool, df func(c gctx.Context, req T, r *Role)) {
	var zero T
	msgType := reflect.TypeOf(zero)
	msgID, ok := pb.GetMsgIDC2S(zero)
	if !ok {
		msgID, ok = pb.GetMsgIDS2S(zero)
	}
	if !ok {
		zap.L().Fatal("Register failed: message type not found in TypeMeta",
			zap.String("type", msgType.String()))
	}

	// 提取出指针底层的结构体 Type (例如 pb.C2SMoveReq)
	elemType := msgType.Elem()

	createFunc := func() pb.VTMessage {
		return reflect.New(elemType).Interface().(pb.VTMessage)
	}

	handleFunc := func(c gctx.Context, msg proto.Message) {
		df(c, msg.(T), c.U.(*Role))
	}

	err := Router().Register(msgID, createFunc, usePool, handleFunc)
	if err != nil {
		zap.L().Fatal("RegisterC2S failed: duplicate register",
			zap.Uint32("msgID", msgID),
			zap.Error(err))
	}
}
