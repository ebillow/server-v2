package main

import (
	"context"
	"server/gateway/logic"
	"server/gateway/session"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"server/pkg/pb"
	"server/pkg/share/app"
	"server/pkg/util"
	"server/pkg/version"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	var a = app.App{
		Init:    Init,
		Action:  Action,
		UnInit:  UnInit,
		OnMsg:   OnServerMsg,
		SrvType: pb.Server_Gateway,
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
	logic.Init()
	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	cfg := loadNetCfg()
	go session.StartWSServer("0.0.0.0:"+util.IToString(flag.TcpPort), cfg)

	zap.S().Infof("listen on ws:%d", flag.TcpPort)
	return nil
}

func UnInit(ctx context.Context) {
	session.Close()
	zap.S().Info("server closed")
}

func loadNetCfg() *session.Config {
	d, err := time.ParseDuration("60s")
	if err != nil {
		zap.L().Error("parse read_dead_line err", zap.Error(err))
		return nil
	}
	cfg := &session.Config{
		ReadDeadline:        d,
		OutChanSize:         16,
		ReadSocketBuffSize:  1024,
		WriteSocketBuffSize: 1024,
		RecvPkgLenLimit:     uint32(10240),
	}
	zap.S().Infof("read_dead_line=%v, out_chan_size=%d, read_sock_size=%d, write_sock_size=%d,  pkg_len_limit=%d",
		cfg.ReadDeadline, cfg.OutChanSize, cfg.ReadSocketBuffSize, cfg.WriteSocketBuffSize, cfg.RecvPkgLenLimit)
	return cfg
}

func OnServerMsg(ctx gctx.Context) {
	if ctx.Flag == gctx.Forward {
		ses := session.Get(ctx.SesID)
		if ses == nil {
			return
		}
		ses.SendBytes(ctx.MsgID, ctx.Data)
		return
	}

	router.S().Handle(ctx)
}
