package logic

import (
	"google.golang.org/protobuf/proto"
	"server/account/logic/login"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

func init() {
	router.C().Msg(msgid.MsgIDC2S_C2SLogin, onLogin)
}

func onLogin(msgBase proto.Message, c gctx.Context) {
	msg := msgBase.(*pb.C2SLogin)
	if msg == nil {
		return
	}

	msgS := &pb.S2SReqLogin{
		Req:   msg,
		SesID: c.Msg.SesID,
	}
	login.Login(msgS)
}
