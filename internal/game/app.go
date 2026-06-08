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
	// inject()

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

func init() {
	role.InjectLoginMgr(&logon_service.Mgr)
	role.InjectCompCreate(&component.Create)
}

func OnServerMsg(c gctx.Context) {
	if c.Head.SesID != 0 { // 客户端消息
		err := role.Mgr.DispatchBySesID(c.Head.SesID, role.Event{
			Ctx: c,
		})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Inline(&c))
		}
		return
	}

	if c.Head.ActorID > uint64(pb.ActorID_IDAccBegin) { // 服务器发来的角色消息
		err := role.Mgr.Dispatch(c.Head.ActorID, role.Event{
			Ctx: c,
		})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Inline(&c))
		}
		return
	}

	if c.Head.ActorID > 0 { // 服务器发来的公共模块消息
		err := actor.Actors.Dispatch(c.Head.ActorID, actor.Event{Ctx: c})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Inline(&c), zap.String("actor", pb.ActorID_name[int32(c.Head.ActorID)]))
		}
		return
	}

	// 不能处理逻辑，只分发
	if err := router.R().Handle(c); err != nil {
		zap.L().Error("Handle", zap.Error(err), zap.Inline(&c))
	}
}
