package main

import (
	"github.com/nats-io/nats.go"
	"server/pkg/gnet/router"
	"server/pkg/pb"
)

func OnServerMsg(natsMsg *pb.NatsMsg, raw *nats.Msg) {
	router.C().Handle(natsMsg, raw)
}
