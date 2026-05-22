package robot

import (
	"context"
	_ "net/http/pprof"
	"server/internal/robot/clinet"
	"server/internal/robot/logic/monitor"
	robot2 "server/internal/robot/logic/robot"
	"sync"

	"go.uber.org/zap"
)

func Init(ctx context.Context) error {
	// begin := int(flag.SvcIndex) * 100000
	// robot2.Setup = &robot2.ServerCfg{
	// 	ServerAddr: Addr,
	// 	Cnt:        Count,
	// 	BeginID:    begin,
	// 	LoginOnly:  false,
	// }
	monitor.Register()
	robot2.RegisteMsgHandle()

	return nil
}

func UnInit(ctx context.Context) {
	zap.S().Info("closing...")
	clinet.Close()
	zap.S().Info("robot exit")
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	zap.S().Info("start run")
	robot2.InitRobots(robot2.Setup.Cnt, robot2.Setup.BeginID)
	return nil
}
