package main

import (
	"context"
	"server/center/role_mgr"
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
		SrvType: pb.Server_Center,
		Init:    Init,
		Action:  Action,
		UnInit:  UnInit,
		OnMsg:   OnServerMsg,
	}
	var rootCmd = &cobra.Command{
		Use:     "", // 默认直接启动，不需要子命令
		Short:   "start center server",
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
	db.MongoUse(flag.IID + "_center")
	role_mgr.Init()
	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	return nil
}

func UnInit(ctx context.Context) {
}

func OnServerMsg(ctx gctx.Context) {
	router.S().Handle(ctx)
}
