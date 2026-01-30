package main

import (
	"github.com/nats-io/nats.go"
	"server/gateway/session"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/router"
)

func OnServerMsg(msg *nats.Msg) {
	sesID := msgq.SessionID(msg)
	if sesID != 0 {
		ses := session.GetCliSession(sesID)
		if ses == nil {
			return
		}
		msgID := msgq.MsgID(msg)
		ses.SendBytes(msgID, msg.Data)
		return
	}

	router.S().Handle(msg)
}
