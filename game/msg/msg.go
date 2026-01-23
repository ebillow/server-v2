package msg

import (
	"github.com/nats-io/nats.go"
	"server/game/role"
	"server/internal/util"
)

func OnClientMsg(msg *nats.Msg) {
	sesID := util.ParseUint64(msg.Header.Get("session"))
	role.RoleMgr().PostEventBySesID(sesID, role.Event{
		Msg:    msg,
		CliMsg: true,
	})
}

func OnServerMsg(msg *nats.Msg) {
	roleID := util.ParseUint64(msg.Header.Get("role"))
	if roleID != 0 {
		role.RoleMgr().PostEvent(roleID, role.Event{
			Msg: msg,
		})
		return
	}
	// 不是发给角色的消息，post到各自功能协程
}
