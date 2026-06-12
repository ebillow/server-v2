package auth

import (
	"server/api/pb"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/router"
)

func init() {
	router.On(onLogin)
	router.On(onClearRole)
}

func onLogin(h gmsg.Head, req *pb.C2SLogin) {
	msgS := &pb.S2SReqLogin{
		Req:   req,
		SesID: h.SesID,
	}
	HandleLoginRequest(msgS)
}

func onClearRole(_ gmsg.Head, req *pb.S2SRoleClear) {
	dispatchEvent(Event{
		Op:    OpRoleClear,
		Clear: req,
	})
}
