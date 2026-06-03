package game

import (
	"context"
	"server/api/pb"
	"server/internal/game/component"
	"server/internal/game/game_db"
	"server/internal/game/logon_service"
	"server/internal/game/role"
	"server/internal/share/actor"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/thread"
	"sync"

	"go.uber.org/zap"
)

func Init(ctx context.Context) error {
	inject()

	db.MongoUse(flag.IID + "_game")
	game_db.CreateIndex()

	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	logon_service.Mgr.Start()
	thread.GoSafe(func() {
		role.Run(ctx)
	})
	return nil
}

func UnInit(ctx context.Context) {
	logon_service.Mgr.Close()
}

func inject() {
	role.InjectLoginMgr(&logon_service.Mgr)
	role.InjectRouter(router.R())
	role.InjectCompCreate(&component.Create)
}

func OnServerMsg(c gctx.Context) {
	if c.SesID != 0 { // 客户端消息
		err := role.Mgr.DispatchBySesID(c.SesID, role.Event{
			Ctx: c,
		})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Inline(&c))
		}
		return
	}

	if c.ActorID > uint64(pb.ActorID_IDAccBegin) { // 服务器发来的角色消息
		err := role.Mgr.Dispatch(c.ActorID, role.Event{
			Ctx: c,
		})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Inline(&c))
		}
		return
	}

	if c.ActorID > 0 { // 服务器发来的公共模块消息
		err := actor.Actors.Dispatch(c.ActorID, actor.Event{Ctx: c})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Inline(&c), zap.String("actor", pb.ActorID_name[int32(c.ActorID)]))
		}
		return
	}

	// 不能处理逻辑，只分发
	if err := router.R().Handle(c); err != nil {
		zap.L().Error("Handle", zap.Error(err), zap.Inline(&c))
	}
}
