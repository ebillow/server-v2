package role

import (
	"reflect"
	"server/api/pb"
	"server/pkg/gnet/gctx"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func OnC2S[T pb.VTMessage](df func(c gctx.Context, req T, r *Role)) {
	onC2S(false, df)
}
func OnC2SUsePool[T pb.VTMessage](df func(c gctx.Context, req T, r *Role)) {
	onC2S(true, df)
}

func OnS2SRole[T pb.VTMessage](df func(c gctx.Context, req T, r *Role)) {
	onS2SWithRole(false, df)
}
func OnS2SRoleUsePool[T pb.VTMessage](df func(c gctx.Context, req T, r *Role)) {
	onS2SWithRole(true, df)
}

func onC2S[T pb.VTMessage](usePool bool, df func(c gctx.Context, req T, r *Role)) {
	var zero T
	msgType := reflect.TypeOf(zero)
	msgID, err := pb.GetMsgIDC2S(zero)
	if err != nil {
		zap.L().Fatal("RegisterC2S failed: message type not found in TypeMeta",
			zap.String("type", msgType.String()),
			zap.Error(err))
	}

	// 提取出指针底层的结构体 Type (例如 pb.C2SMoveReq)
	elemType := msgType.Elem()

	createFunc := func() pb.VTMessage {
		return reflect.New(elemType).Interface().(pb.VTMessage)
	}

	handleFunc := func(c gctx.Context, msg proto.Message) {
		// 框架层直接进行 O(1) 的类型断言 (Type Assertion)，没有任何性能损耗
		df(c, msg.(T), c.U.(*Role))
	}

	err = cRouter().Register(msgID, createFunc, usePool, handleFunc)
	if err != nil {
		zap.L().Fatal("RegisterC2S failed: duplicate register",
			zap.Uint32("msgID", msgID),
			zap.Error(err))
	}
}

func onS2SWithRole[T pb.VTMessage](usePool bool, df func(c gctx.Context, req T, r *Role)) {
	var zero T
	msgType := reflect.TypeOf(zero)
	msgID, err := pb.GetMsgIDS2S(zero)
	if err != nil {
		zap.L().Fatal("RegisterC2S failed: message type not found in TypeMeta",
			zap.String("type", msgType.String()),
			zap.Error(err))
	}

	// 提取出指针底层的结构体 Type (例如 pb.C2SMoveReq)
	elemType := msgType.Elem()

	createFunc := func() pb.VTMessage {
		return reflect.New(elemType).Interface().(pb.VTMessage)
	}

	handleFunc := func(c gctx.Context, msg proto.Message) {
		// 框架层直接进行 O(1) 的类型断言 (Type Assertion)，没有任何性能损耗
		df(c, msg.(T), c.U.(*Role))
	}

	err = sRouter().Register(msgID, createFunc, usePool, handleFunc)
	if err != nil {
		zap.L().Fatal("RegisterC2S failed: duplicate register",
			zap.Uint32("msgID", msgID),
			zap.Error(err))
	}
}
