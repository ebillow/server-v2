package logic

import (
	"server/api/pb"
	"server/internal/gateway/session"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
)

func init() {
	router.On(onLoginSuccess)
	router.On(onDisconnect)
}

func onLoginSuccess(c gctx.Context, req *pb.S2SResLogin) {
	ses := session.Get(c.SesID)
	if ses == nil {
		return
	}
	ses.UpdateSerId(req.GameID)
	ses.Send(req.Res)
}

func onDisconnect(c gctx.Context, req *pb.S2SS2GtDisconnect) {
	ses := session.Get(c.SesID)
	if ses == nil {
		return
	}

	ses.Close(req.Why)
}
