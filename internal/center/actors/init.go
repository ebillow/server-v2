package actors

import (
	"server/api/pb"
	"server/internal/center/actors/example"
	"server/internal/center/actors/global"
	"server/internal/share/actor"

	"go.uber.org/zap"
)

func Create() {
	if err := actor.Actors.Init(uint64(pb.ActorID_IDExample), &example.Example{}, 0); err != nil {
		zap.L().Error("init example failed", zap.Error(err))
	}
	if err := actor.Actors.Init(uint64(pb.ActorID_IDGlobal), &global.Global{}, 0); err != nil {
		zap.L().Error("init example failed", zap.Error(err))
	}
}
