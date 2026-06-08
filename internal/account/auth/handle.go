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

func onLogin(h gctx.Head, req *pb.C2SLogin) {
	msgS := &pb.S2SReqLogin{
		Req:   req,
		SesID: h.SesID,
	}
	HandleLoginRequest(msgS)
}

func onClearRole(_ gctx.Head, req *pb.S2SRoleClear) {
	dispatchEvent(Event{
		Op:    OpRoleClear,
		Clear: req,
	})
}
