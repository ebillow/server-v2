package main

import (
	"context"
	"github.com/spf13/cobra"
	_ "net/http/pprof"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/version"
	"server/robot/clinet"
	"server/robot/logic/robot"
	"sync"
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Robot,
		Init:    Init,
		Action:  Action,
		UnInit:  UnInit,
	}
	var rootCmd = &cobra.Command{
		Use:     "", // 默认直接启动，不需要子命令
		Short:   "start robot",
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
	robot.Setup = &robot.ServerCfg{
		ServerAddr: "127.0.0.1:3001",
		Cnt:        1000,
		BeginID:    1,
		LoginOnly:  false,
	}
	// err := component.ReadJson(component.Setup, "./setup.json")
	// if err != nil {
	// 	return err
	// }

	robot.RegisteMsgHandle()
	return nil
}

func UnInit(ctx context.Context) {
	logger.Info("closing...")
	clinet.Close()
	logger.Info("robot exit")
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	logger.Info("start run")
	robot.InitRobots(robot.Setup.Cnt, robot.Setup.BeginID)

	return nil
}
