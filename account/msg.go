package main

import (
	"github.com/nats-io/nats.go"
	"server/pkg/gnet/router"
)

func OnServerMsg(msg *nats.Msg) {
	router.C().Handle(msg)
}
