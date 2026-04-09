package main

import (
	"context"
	_ "net/http/pprof"
	"server/account/acc_db"
	"server/account/logic"
	"server/account/logic/login"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/version"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Account,
		Init:    Init,
		Action:  Action,
		UnInit:  UnInit,
		OnMsg:   OnServerMsg,
	}
	var rootCmd = &cobra.Command{
		Use:     "", // 默认直接启动，不需要子命令
		Short:   "start account server",
		Run:     a.RootCmdRun,
		Version: version.String(),
	}

	rootCmd.AddCommand(
		version.CobraCmd(), // 打印version
	)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func Init(ctx context.Context) error {
	logic.Init()

	db.MongoUse(flag.IID + "_account")
	acc_db.CreateIndex()

	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	login.Start(ctx)
	return nil
}

func UnInit(ctx context.Context) {

}

func OnServerMsg(natsMsg *pb.NatsMsg, raw *nats.Msg) {
	if natsMsg.SerType == pb.Server_Gateway {
		router.C().HandleG(natsMsg, raw)
	} else {
		router.S().HandleG(natsMsg, raw)
	}
}
