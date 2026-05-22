package main

import (
	"server/api/pb"
	"server/internal/center"
	"server/internal/share/app"
	"server/pkg/flag"
	"server/pkg/version"

	"github.com/spf13/cobra"
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Center,
		Init:    center.Init,
		Action:  center.Action,
		UnInit:  center.UnInit,
		OnMsg:   center.OnServerMsg,
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
