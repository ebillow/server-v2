package main

import (
	"context"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"server/game/component"
	"server/game/game_db"
	"server/game/role"
	"server/game/role/login_mgr"
	_ "server/game/role/msg"
	"server/game/role/role_mgr"
	"server/pkg/db"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/version"
	"sync"
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

	rootCmd.AddCommand(
		version.CobraCmd(), // 打印version
	)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func Init(ctx context.Context) error {
	inject()

	db.MongoUse("game")
	game_db.CreateIndex()

	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	login_mgr.Mgr.Start()
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

func OnServerMsg(natMsg *pb.NatsMsg, raw *nats.Msg) {
	if natMsg.RoleID != 0 {
		role.RoleMgr().PostEvent(natMsg.RoleID, role.Event{
			Raw:    raw,
			NatMsg: natMsg,
		})
		return
	}

	if natMsg.SesID != 0 {
		role.RoleMgr().PostEventBySesID(natMsg.SesID, role.Event{
			Raw:    raw,
			NatMsg: natMsg,
			CliMsg: true,
		})
		return
	}

	router.S().HandleG(natMsg, raw)
}
