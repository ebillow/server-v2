package logic

import (
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/gateway/session"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"

	"google.golang.org/protobuf/proto"
)

func init() {
	router.S().OnG(msgid.MsgIDS2S_S2SResLogin, onLoginSuccess)
	router.S().OnG(msgid.MsgIDS2S_S2SS2GtDisconnect, onDisconnect)
}

func onLoginSuccess(c gctx.Context, msgBase proto.Message) {
	msg := msgBase.(*pb.S2SResLogin)
	ses := session.Get(c.SesID)
	if ses == nil {
		return
	}
	ses.UpdateSerId(msg.GameID)
	ses.Send(msg.Res)
}

func onDisconnect(c gctx.Context, msgBase proto.Message) {
	ses := session.Get(c.SesID)
	if ses == nil {
		return
	}

	msg := msgBase.(*pb.S2SS2GtDisconnect)
	ses.Close(msg.Why)
}
