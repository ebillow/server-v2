package main

import (
	"server/api/pb"
	"server/internal/gateway"
	"server/internal/share/app"
	"server/pkg/flag"
	"server/pkg/version"

	"github.com/spf13/cobra"
)

func main() {
	var a = app.App{
		Init:    gateway.Init,
		Action:  gateway.Action,
		UnInit:  gateway.UnInit,
		OnMsg:   gateway.OnServerMsg,
		SrvType: pb.Server_Gateway,
	}
	var rootCmd = &cobra.Command{
		Use:     "", // 默认直接启动，不需要子命令
		Short:   "start gateway server",
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
