package main

import (
	"github.com/nats-io/nats.go"
	"server/game/role"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/router"
)

func OnServerMsg(msg *nats.Msg) {
	roleID := msgq.RoleID(msg)
	if roleID != 0 {
		role.RoleMgr().PostEvent(roleID, role.Event{
			Msg: msg,
		})
		return
	}

	sesID := msgq.SessionID(msg)
	if sesID != 0 {
		role.RoleMgr().PostEventBySesID(sesID, role.Event{
			Msg:    msg,
			CliMsg: true,
		})
		return
	}

	router.S().Handle(msg)
}
