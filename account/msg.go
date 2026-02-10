package main

import (
	"github.com/nats-io/nats.go"
	"server/pkg/gnet/router"
	"server/pkg/pb"
)

func OnServerMsg(natsMsg *pb.NatsMsg, raw *nats.Msg) {
	if natsMsg.SerType == pb.Server_Gateway {
		router.C().Handle(natsMsg, raw)
	} else {
		router.S().Handle(natsMsg, raw)
	}
}
