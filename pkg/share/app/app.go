package app

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"path/filepath"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/ghttp"
	"server/pkg/gnet/msgq"
	"server/pkg/idgen"
	"server/pkg/lock"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/thread"
	"server/pkg/util"
	"server/pkg/version"
	"sync"
)

type App struct {
	Init    func(ctx context.Context) error
	Action  func(ctx context.Context, wait *sync.WaitGroup) error
	UnInit  func(ctx context.Context)
	OnMsg   func(*nats.Msg)
	SrvType pb.Server
}

func (a *App) RootCmdRun(cmd *cobra.Command, args []string) {
	cmd.Flags().SortFlags = false
	flag.Init(a.SrvType, cmd.PersistentFlags(), false)
	flag.Debug(cmd.PersistentFlags())

	ctx, cancel := context.WithCancel(context.Background())
	var wait sync.WaitGroup

	if err := a.init(ctx); err != nil {
		panic(err)
	}
	if err := a.action(ctx, &wait); err != nil {
		panic(err)
	}
	cancel()
	_ = a.unInit(ctx)
	wait.Wait()
}

func (a *App) init(ctx context.Context) error {
	idgen.Init(flag.SvcIndex)
	a.initLog()
	version.LogVersion()
	if err := a.initDB(); err != nil {
		panic(err)
	}
	lock.InitPool(db.Redis)

	if err := msgq.Q.Init("nats://localhost:4222", a.SrvName(), int32(flag.SvcIndex), nats.UserInfo("123456", "123456")); err != nil {
		return err
	}

	if err := a.Init(ctx); err != nil {
		return err
	}

	return nil
}

func (a *App) SrvName() string {
	return flag.SrvName(flag.SrvType)
}

func (a *App) initLog() {
	filePath := filepath.Join("./bin/logs", fmt.Sprintf("%s_%d.logger", a.SrvName(), flag.SvcIndex))
	logger.NewZapLog(filePath, logger.Config{
		Level:        -1,
		Console:      true,
		ConsoleColor: true,
	})
}

func (a *App) initDB() error {
	if err := db.InitRedis(db.RedisCfg{
		Addr:     []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
		Password: "",
		DB:       0,
	}, 0); err != nil {
		return err
	}

	if err := db.InitMongo(db.MongoCfg{
		URI:    "mongodb://localhost:27017",
		DbName: "game",
	}, 16, 32); err != nil {
		return err
	}
	return nil
}

func (a *App) action(ctx context.Context, wait *sync.WaitGroup) error {
	if err := a.Action(ctx, wait); err != nil {
		return err
	}

	ghttp.Start(ctx, wait, flag.HttpPort)
	if a.OnMsg != nil {
		if err := msgq.Q.Serve(a.OnMsg); err != nil {
			return err
		}
	}
	flag.SetReady()
	fmt.Print(util.SuccessShow)
	zap.L().Info("启动成功", zap.String("version", version.GitCommit))

	thread.WaitExit()

	return nil
}

func (a *App) unInit(ctx context.Context) error {
	a.UnInit(ctx)
	// zap.L().Info("closing...")
	_ = db.CloseMongo()
	zap.L().Info("server exit")
	return nil
}
