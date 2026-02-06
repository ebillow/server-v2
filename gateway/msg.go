package main

import (
	"github.com/nats-io/nats.go"
	"server/gateway/session"
	"server/pkg/gnet/gctx"
	"server/pkg/pb"
)

func OnServerMsg(natsMsg *pb.NatsMsg, raw *nats.Msg) {
	if natsMsg.SesID != 0 {
		ses := session.GetCliSession(natsMsg.SesID)
		if ses == nil {
			return
		}
		ses.Post(gctx.Context{
			U:   ses,
			Raw: raw,
			Msg: natsMsg,
		})
		return
	}
}
