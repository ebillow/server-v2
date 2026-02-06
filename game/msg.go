package main

import (
	"github.com/nats-io/nats.go"
	"server/game/role"
	"server/pkg/gnet/router"
	"server/pkg/pb"
)

func OnServerMsg(natMsg *pb.NatsMsg, raw *nats.Msg) {
	if natMsg.RoleID != 0 {
		role.RoleMgr().PostEvent(natMsg.RoleID, role.Event{
			Raw:    raw,
			NatMsg: natMsg,
		})
		return
	}

	if natMsg.SesID != 0 {
		role.RoleMgr().PostEventBySesID(natMsg.SesID, role.Event{
			Raw:    raw,
			NatMsg: natMsg,
			CliMsg: true,
		})
		return
	}

	router.S().Handle(natMsg, raw)
}
