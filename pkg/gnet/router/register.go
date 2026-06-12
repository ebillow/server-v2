package router

import (
	"reflect"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/gerror"
	"server/pkg/gnet/pkg"

	"go.uber.org/zap"
)

var (
	msgRouter = NewMsgRouter(int32(msgid.MsgIDMax_C2SMax))
)

func R() *Router {
	return msgRouter
}

func On[T pb.VTMessage](df func(h pkg.Head, req T)) {
	register(false, df)
}

// OnP 消息处理，使用对象池，req不能传递到其它协程，不能持有
func OnP[T pb.VTMessage](df func(h pkg.Head, req T)) {
	register(true, df)
}

func OnRpc[Req pb.VTMessage, Res pb.VTMessage](df func(h pkg.Head, req Req, res Res)) {
	registerRpc(false, df)
}

// OnRpcP 消息处理，使用对象池，req, res不能传递到其它协程，不能持有
func OnRpcP[Req pb.VTMessage, Res pb.VTMessage](df func(h pkg.Head, req Req, res Res)) {
	registerRpc(true, df)
}

func register[T pb.VTMessage](usePool bool, df func(h pkg.Head, req T)) {
	var req T
	msgID, createFunc, err := FindMsgIDAndCreateFunc(req)
	if err != nil {
		zap.L().Error("register fail", zap.Error(err))
		return
	}
	handleFunc := func(p pkg.Packet, msg pb.VTMessage) {
		df(p.Head, msg.(T))
	}

	err = R().Register(msgID, createFunc, usePool, handleFunc)
	if err != nil {
		zap.L().Fatal("register failed: duplicate register",
			zap.String("msgType", reflect.TypeOf(req).String()),
			zap.Uint32("msgID", msgID),
			zap.Error(err))
	}
}

func registerRpc[Req pb.VTMessage, Res pb.VTMessage](usePool bool, df func(h pkg.Head, req Req, res Res)) {
	var req Req

	msgID, reqCreate, err := FindMsgIDAndCreateFunc(req)
	if err != nil {
		zap.L().Error("register fail", zap.Error(err))
		return
	}

	var res Res
	resType := reflect.TypeOf(res)
	resElemType := resType.Elem()
	resCreate := func() pb.VTMessage {
		return reflect.New(resElemType).Interface().(pb.VTMessage)
	}

	handleFunc := func(p pkg.Packet, req pb.VTMessage, res pb.VTMessage) {
		df(p.Head, req.(Req), res.(Res))
	}

	err = R().RegisterRpc(msgID, reqCreate, resCreate, usePool, handleFunc)
	if err != nil {
		zap.L().Fatal("Register failed: duplicate register",
			zap.String("msgType", reflect.TypeOf(req).String()),
			zap.Uint32("msgID", msgID),
			zap.Error(err))
	}
}

func FindMsgIDAndCreateFunc[T pb.VTMessage](req T) (uint32, func() pb.VTMessage, error) {
	msgType := reflect.TypeOf(req)
	msgID, ok := pb.GetMsgIDS2S(req)
	isCli := false
	if !ok {
		msgID, ok = pb.GetMsgIDC2S(req)
		isCli = true
	}
	if !ok {
		return msgID, nil, gerror.Newf("message %s not found in TypeMeta", msgType.String())
	}

	var createFunc func() pb.VTMessage
	if isCli {
		createFunc = pb.NewFuncS2C(msgid.MsgIDS2C(msgID))
	} else {
		createFunc = pb.NewFuncS2S(msgid.MsgIDS2S(msgID))
	}
	if createFunc == nil {
		return msgID, nil, gerror.Newf("message %s not found create func in TypeMeta", msgType.String())
	}
	return msgID, createFunc, nil
}
