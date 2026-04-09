package logic

import (
	session "server/gateway/session/v2"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/pb/msgid"

	"google.golang.org/protobuf/proto"
)

func init() {
	router.S().OnG(msgid.MsgIDS2S_S2SResLogin, onLoginSuccess)
	router.S().OnG(msgid.MsgIDS2S_S2SS2GtDisconnect, onDisconnect)
}

func onLoginSuccess(c gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.S2SResLogin)
	ses := session.GetSession(c.Msg.SesID)
	if ses == nil {
		return
	}
	ses.UpdateSerId(msg.GameID)
	ses.Send(msg.Res)
}

func onDisconnect(c gctx.Context, msgBase proto.Message) {
	ses := session.GetSession(c.Msg.SesID)
	if ses == nil {
		return
	}

	msg := msgBase.(*pb.S2SS2GtDisconnect)
	ses.Close(msg.Why)
}
