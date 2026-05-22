package main

import (
	"server/api/pb"
	"server/internal/account"
	"server/internal/share/app"
	"server/pkg/flag"
	"server/pkg/version"

	"github.com/spf13/cobra"
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Account,
		Init:    account.Init,
		Action:  account.Action,
		UnInit:  account.UnInit,
		OnMsg:   account.OnServerMsg,
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
