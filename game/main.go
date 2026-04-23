package main

import (
	"context"
	"server/game/component"
	"server/game/game_db"
	"server/game/role"
	"server/game/role/login_mgr"
	_ "server/game/role/msg"
	"server/game/role/role_mgr"
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
	login_mgr.Mgr.Start()
	thread.GoSafe(func() {
		role_mgr.Run(ctx)
	})
	return nil
}

func UnInit(ctx context.Context) {
	login_mgr.Mgr.Close()
}

func inject() {
	role.InjectLoginMgr(&login_mgr.Mgr)
	role.InjectRoleMgr(role_mgr.Mgr)
	role.InjectCRouter(router.C())
	role.InjectSRouter(router.S())

	role.CreateComps = component.CreateComps
}

func OnServerMsg(ctx gctx.Context) {
	if ctx.RoleID != 0 {
		role.RoleMgr().PostEvent(ctx.RoleID, role.Event{
			Ctx: ctx,
		})
		return
	}

	if ctx.SesID != 0 {
		role.RoleMgr().PostEventBySesID(ctx.SesID, role.Event{
			Ctx:    ctx,
			CliMsg: true,
		})
		return
	}

	router.S().Handle(ctx)
}
