package robot

import (
	"context"
	_ "net/http/pprof"
	"server/internal/robot/clinet"
	"server/internal/robot/logic/robot"
	"server/pkg/flag"
	"sync"

	"go.uber.org/zap"
)

func Init(ctx context.Context) error {
	begin := int(flag.SvcIndex) * 100000
	robot.Setup = &robot.ServerCfg{
		ServerAddr: "127.0.0.1:30001",
		Cnt:        10000,
		BeginID:    begin,
		LoginOnly:  false,
	}
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
