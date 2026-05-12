package main

import (
	"context"
	_ "net/http/pprof"
	"server/pkg/flag"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/version"
	"server/robot/clinet"
	"server/robot/logic/monitor"
	"server/robot/logic/robot"
	"sync"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Robot,
		Init:    Init,
		Action:  Action,
		UnInit:  UnInit,
	}
	var rootCmd = &cobra.Command{
		Use:   "", // 默认直接启动，不需要子命令
		Short: "start robot",
		// PreRun:  PreRun,
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

var (
	Count int
	Addr  string
)

func Init(ctx context.Context) error {
	begin := int(flag.SvcIndex) * 100000
	robot.Setup = &robot.ServerCfg{
		ServerAddr: Addr,
		Cnt:        Count,
		BeginID:    begin,
		LoginOnly:  false,
	}
	monitor.Register()
	robot.RegisteMsgHandle()

	return nil
}

func UnInit(ctx context.Context) {
	zap.S().Info("closing...")
	clinet.Close()
	zap.S().Info("robot exit")
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	zap.S().Info("start run")
	robot.InitRobots(robot.Setup.Cnt, robot.Setup.BeginID)
	return nil
}
