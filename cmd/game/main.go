package main

import (
	"server/api/pb"
	"server/internal/game"
	"server/internal/share/app"
	"server/pkg/flag"
	"server/pkg/version"

	"github.com/spf13/cobra"
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Game,
		Init:    game.Init,
		Action:  game.Action,
		UnInit:  game.UnInit,
		OnMsg:   game.OnServerMsg,
	}
	var rootCmd = &cobra.Command{
		Use:     "", // 默认直接启动，不需要子命令
		Short:   "start game server",
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
