package game

import (
	"context"
	"server/api/pb"
	"server/internal/game/component"
	"server/internal/game/game_db"
	role2 "server/internal/game/role"
	"server/internal/game/role/logon_service"
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
		role2.Run(ctx)
	})
	return nil
}

func UnInit(ctx context.Context) {
	logon_service.Mgr.Close()
}

func inject() {
	role2.InjectLoginMgr(&logon_service.Mgr)
	role2.InjectCRouter(router.C())
	role2.InjectSRouter(router.S())
	role2.InjectCompCreate(&component.Create)
}

func OnServerMsg(ctx gctx.Context) {
	if ctx.ActorID > uint64(pb.ActorID_IDAccBegin) { // 服务器发来的角色消息
		err := role2.Mgr.Dispatch(ctx.ActorID, role2.Event{
			Ctx: ctx,
		})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Uint64("role", ctx.ActorID))
		}
		return
	}

	if ctx.ActorID > 0 { // 服务器发来的公共模块消息
		err := actor.Actors.Post(ctx.ActorID, actor.Event{Ctx: ctx})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Uint64("actor", ctx.ActorID), zap.String("actor", pb.ActorID_name[int32(ctx.ActorID)]))
		}
		return
	}

	if ctx.SesID != 0 { // 客户端消息
		err := role2.Mgr.DispatchBySesID(ctx.SesID, role2.Event{
			Ctx: ctx,
		})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Uint64("session", ctx.SesID))
		}
		return
	}

	router.S().Handle(ctx) // todo 丢到协程里处理
}
