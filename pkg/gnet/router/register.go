package router

import (
	"reflect"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/gerror"
	"server/pkg/gnet/gmsg"
)

var (
	msgRouter = NewMsgRouter(int32(msgid.MsgIDMax_C2SMax))
)

func R() *Router {
	return msgRouter
}

// On 注册消息处理
func On[T pb.VTMessage](df func(req T)) {
	register(false, df)
}

// OnWithHead 注册消息处理，消息处理中带Head信息
func OnWithHead[T pb.VTMessage](df func(h gmsg.Head, req T)) {
	registerWithHead(false, df)
}

// OnP 消息处理，使用对象池，req不能传递到其它协程，不能持有
func OnP[T pb.VTMessage](df func(req T)) {
	register(true, df)
}

// OnRpc Rpc消息处理,收到Req返回Res
func OnRpc[Req pb.VTMessage, Res pb.VTMessage](df func(req Req, res Res)) {
	registerRpc(false, df)
}

// OnRpcP 消息处理，使用对象池，req, res不能传递到其它协程，不能持有
func OnRpcP[Req pb.VTMessage, Res pb.VTMessage](df func(req Req, res Res)) {
	registerRpc(true, df)
}

func register[T pb.VTMessage](usePool bool, df func(req T)) {
	var req T
	msgID, createFunc, err := FindMsgIDAndCreateFunc(req)
	if err != nil {
		panic(err)
		return
	}
	handleFunc := func(p gmsg.Message, msg pb.VTMessage) {
		df(msg.(T))
	}

	err = R().Register(msgID, createFunc, usePool, handleFunc)
	if err != nil {
		panic(err)
	}
}

func registerWithHead[T pb.VTMessage](usePool bool, df func(h gmsg.Head, req T)) {
	var req T
	msgID, createFunc, err := FindMsgIDAndCreateFunc(req)
	if err != nil {
		panic(err)
		return
	}
	handleFunc := func(p gmsg.Message, msg pb.VTMessage) {
		df(p.Head, msg.(T))
	}

	err = R().Register(msgID, createFunc, usePool, handleFunc)
	if err != nil {
		panic(err)
	}
}

func registerRpc[Req pb.VTMessage, Res pb.VTMessage](usePool bool, df func(req Req, res Res)) {
	var req Req

	msgID, reqCreate, err := FindMsgIDAndCreateFunc(req)
	if err != nil {
		panic(err)
		return
	}

	var res Res
	resType := reflect.TypeOf(res)
	resElemType := resType.Elem()
	resCreate := func() pb.VTMessage { // todo 注册到pb.help
		return reflect.New(resElemType).Interface().(pb.VTMessage)
	}

	handleFunc := func(p gmsg.Message, req pb.VTMessage, res pb.VTMessage) {
		df(req.(Req), res.(Res))
	}

	err = R().RegisterRpc(msgID, reqCreate, resCreate, usePool, handleFunc)
	if err != nil {
		panic(err)
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
		createFunc = pb.NewFuncC2S(msgid.MsgIDC2S(msgID))
	} else {
		createFunc = pb.NewFuncS2S(msgid.MsgIDS2S(msgID))
	}
	if createFunc == nil {
		return msgID, nil, gerror.Newf("message %s not found create func in TypeMeta", msgType.String())
	}
	return msgID, createFunc, nil
}
