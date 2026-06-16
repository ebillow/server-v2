package role

import (
	"reflect"
	"server/api/pb"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/router"
)

func On[T pb.VTMessage](df func(req T, r *Role)) {
	register(false, df)
}

// OnP 消息处理，使用对象池，req不能传递到其它协程，不能持有
func OnP[T pb.VTMessage](df func(req T, r *Role)) {
	register(true, df)
}

func OnRpc[Req pb.VTMessage, Res pb.VTMessage](df func(req Req, res Res, r *Role)) {
	registerRpc(false, df)
}

// OnRpcP 消息处理，使用对象池，req, res不能传递到其它协程，不能持有
func OnRpcP[Req pb.VTMessage, Res pb.VTMessage](df func(req Req, res Res, r *Role)) {
	registerRpc(true, df)
}

func register[T pb.VTMessage](usePool bool, df func(req T, r *Role)) {
	var req T

	msgID, reqCreate, err := router.FindMsgIDAndCreateFunc(req)
	if err != nil {
		panic(err)
		return
	}

	handleFunc := func(c gmsg.Message, msg pb.VTMessage) {
		df(msg.(T), c.U.(*Role))
	}

	err = router.R().Register(msgID, reqCreate, usePool, handleFunc)
	if err != nil {
		panic(err)
		return
	}
}

func registerRpc[Req pb.VTMessage, Res pb.VTMessage](usePool bool, df func(req Req, res Res, r *Role)) {
	var req Req

	msgID, reqCreate, err := router.FindMsgIDAndCreateFunc(req)
	if err != nil {
		panic(err)
		return
	}

	var res Res
	resType := reflect.TypeOf(res)
	resElemType := resType.Elem()
	resCreate := func() pb.VTMessage {
		return reflect.New(resElemType).Interface().(pb.VTMessage)
	}

	handleFunc := func(p gmsg.Message, req pb.VTMessage, res pb.VTMessage) {
		df(req.(Req), res.(Res), p.U.(*Role))
	}

	err = router.R().RegisterRpc(msgID, reqCreate, resCreate, usePool, handleFunc)
	if err != nil {
		panic(err)
	}
}
