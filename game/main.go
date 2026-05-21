package main

import (
	"context"
	"server/game/component"
	"server/game/game_db"
	"server/game/role"
	"server/game/role/logon_service"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/thread"
	"server/pkg/version"
	"sync"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Game,
		Init:    Init,
		Action:  Action,
		UnInit:  UnInit,
		OnMsg:   OnServerMsg,
	}
	var rootCmd = &cobra.Command{
		Use:     "", // 默认直接启动，不需要子命令
		Short:   "start game server",
		Run:     a.RootCmdRun,
		Version: version.String(),
	}

	rootCmd.Flags().SortFlags = false
	flag.Init(a.SrvType, rootCmd.PersistentFlags())

	rootCmd.AddCommand(
		version.CobraCmd(), // 打印version
	)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

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
		err := app.Actors.Post(ctx.ActorID, app.Event{Ctx: ctx})
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
