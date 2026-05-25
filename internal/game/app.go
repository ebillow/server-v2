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
	role.InjectCRouter(router.C())
	role.InjectSRouter(router.S())
	role.InjectCompCreate(&component.Create)
}

func OnServerMsg(ctx gctx.Context) {
	if ctx.ActorID > uint64(pb.ActorID_IDAccBegin) { // 服务器发来的角色消息
		err := role.Mgr.Dispatch(ctx.ActorID, role.Event{
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
		err := role.Mgr.DispatchBySesID(ctx.SesID, role.Event{
			Ctx: ctx,
		})
		if err != nil {
			zap.L().Error("PostEvent", zap.Error(err), zap.Uint64("session", ctx.SesID))
		}
		return
	}

	router.S().Handle(ctx) // todo 丢到协程里处理
}
