package main

import (
	"context"
	"github.com/spf13/cobra"
	_ "net/http/pprof"
	"server/account/acc_db"
	"server/account/logic"
	"server/account/logic/login"
	"server/pkg/db"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/version"
	"sync"
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
	db.MongoUse("account")
	acc_db.CreateAccIndex()
	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	login.Start(ctx)
	return nil
}

func UnInit(ctx context.Context) {

}
