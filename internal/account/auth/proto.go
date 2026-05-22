package auth

import (
	"server/api/pb"
	msgid2 "server/api/pb/msgid"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"

	"google.golang.org/protobuf/proto"
)

func init() {
	router.C().OnG(msgid2.MsgIDC2S_C2SLogin, onLogin)

	router.S().OnG(msgid2.MsgIDS2S_S2SRoleClear, onClearRole)
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
	Login(msgS)
}

func onClearRole(_ gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.S2SRoleClear)
	PostEvt(Event{
		Op:    OpRoleClear,
		Clear: msg,
	})
}
