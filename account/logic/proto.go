package logic

import (
	"server/account/logic/login"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/pb/msgid"

	"google.golang.org/protobuf/proto"
)

func init() {
	router.C().OnG(msgid.MsgIDC2S_C2SLogin, onLogin)

	router.S().OnG(msgid.MsgIDS2S_S2SRoleClear, onClearRole)
}

func onLogin(c gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.C2SLogin)
	if msg == nil {
		return
	}

	msgS := &pb.S2SReqLogin{
		Req:   msg,
		SesID: c.SesID,
	}
	login.Login(msgS)
}

func onClearRole(_ gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.S2SRoleClear)
	login.PostEvt(login.EvtParam{
		Op:    login.OpRoleClear,
		Clear: msg,
	})
}
