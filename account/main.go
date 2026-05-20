package main

import (
	"context"
	_ "net/http/pprof"
	"server/account/acc_db"
	"server/account/logic"
	"server/account/logic/auth"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/version"
	"sync"

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

	rootCmd.Flags().SortFlags = false
	flag.Init(a.SrvType, rootCmd.PersistentFlags())

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
	auth.Start(ctx)
	return nil
}

func UnInit(ctx context.Context) {

}

func OnServerMsg(ctx gctx.Context) {
	if ctx.FromSer == uint8(pb.Server_Gateway) {
		router.C().Handle(ctx)
	} else {
		router.S().Handle(ctx)
	}
}
