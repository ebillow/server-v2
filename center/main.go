package main

import (
	"context"
	"server/center/actors"
	"server/center/onlines"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/msgq"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/version"
	"sync"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	var a = app.App{
		SrvType: pb.Server_Center,
		Init:    Init,
		Action:  Action,
		UnInit:  UnInit,
		OnMsg:   OnServerMsg,
	}
	var rootCmd = &cobra.Command{
		Use:     "", // 默认直接启动，不需要子命令
		Short:   "start center server",
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
	db.MongoUse(flag.IID + "_center")
	actors.Create()

	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	app.Actors.Run()
	return nil
}

func UnInit(ctx context.Context) {
	app.Actors.StopAndWait()
}

func OnServerMsg(ctx gctx.Context) {
	if ctx.Flag == gctx.Forward {
		gameID, ok := onlines.GetGameID(ctx.ActorID)
		if ok {
			msgq.Q.Send(pb.Server(ctx.ToSer), gameID, ctx.MsgID, ctx.Data, ctx.ActorID, ctx.SesID)
		}
	} else if ctx.ActorID > 0 {
		err := app.Actors.Post(ctx.ActorID, app.Event{Ctx: ctx})
		if err != nil {
			zap.L().Error("pos msg error", zap.Error(err))
		}
	} else {
		err := app.Actors.Post(uint64(pb.ActorID_IDGlobal), app.Event{Ctx: ctx})
		if err != nil {
			zap.L().Error("pos msg error", zap.Error(err))
		}
	}
}
