package logic

import (
	"server/center/logic/example"
	"server/center/logic/global"
	"server/pkg/pb"
	"server/pkg/share/app"

	"go.uber.org/zap"
)

func Create() {
	if err := app.Actors.Init(uint64(pb.ActorID_IDExample), &example.Example{}, 0); err != nil {
		zap.L().Error("init example failed", zap.Error(err))
	}
	if err := app.Actors.Init(uint64(pb.ActorID_IDGlobal), &global.Global{}, 0); err != nil {
		zap.L().Error("init example failed", zap.Error(err))
	}
}
