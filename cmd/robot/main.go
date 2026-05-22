package main

import (
	"server/api/pb"
	"server/internal/robot"
	"server/internal/share/app"
	"server/pkg/flag"
	"server/pkg/version"

	"github.com/spf13/cobra"
)

var (
	Count int
	Addr  string
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Robot,
		Init:    robot.Init,
		Action:  robot.Action,
		UnInit:  robot.UnInit,
	}
	var rootCmd = &cobra.Command{
		Use:     "", // 默认直接启动，不需要子命令
		Short:   "start robot",
		Run:     a.RootCmdRun,
		Version: version.String(),
	}

	rootCmd.Flags().SortFlags = false
	fs := rootCmd.PersistentFlags()
	fs.StringVar(&Addr, "addr", "127.0.0.1:30001", "gateway addr")
	fs.IntVar(&Count, "count", 1, "数量")
	flag.Init(a.SrvType, fs)

	rootCmd.AddCommand(
		version.CobraCmd(), // 打印version
	)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
