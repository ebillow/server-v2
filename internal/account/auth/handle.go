package auth

import (
	"server/api/pb"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
)

func init() {
	router.On(onLogin)
	router.On(onClearRole)
}

func onLogin(c gctx.Context, req *pb.C2SLogin) {
	msgS := &pb.S2SReqLogin{
		Req:   req,
		SesID: c.SesID,
	}
	HandleLoginRequest(msgS)
}

func onClearRole(_ gctx.Context, req *pb.S2SRoleClear) {
	dispatchEvent(Event{
		Op:    OpRoleClear,
		Clear: req,
	})
}
