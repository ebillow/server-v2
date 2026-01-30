package logic

import (
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"server/account/logic/login"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

func init() {
	router.C().Msg(msgid.MsgIDC2S_C2SLogin, onLogin)
}

func onLogin(msgBase proto.Message, qm *nats.Msg) {
	msg := msgBase.(*pb.C2SLogin)
	if msg == nil {
		return
	}

	msgS := &pb.S2SReqLogin{
		Req:   msg,
		SesID: msgq.SessionID(qm),
	}
	login.Login(msgS)
}
