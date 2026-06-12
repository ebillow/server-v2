package logic

import (
	"server/api/pb"
	"server/internal/gateway/session"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/router"
)

func init() {
	router.On(onLoginSuccess)
	router.On(onDisconnect)
}

func onLoginSuccess(h gmsg.Head, req *pb.S2SResLogin) {
	ses := session.Get(h.SesID)
	if ses == nil {
		return
	}
	ses.UpdateSerId(req.GameID)
	ses.Send(req.Res)
}

func onDisconnect(h gmsg.Head, req *pb.S2SS2GtDisconnect) {
	ses := session.Get(h.SesID)
	if ses == nil {
		return
	}

	ses.Close(req.Why)
}
