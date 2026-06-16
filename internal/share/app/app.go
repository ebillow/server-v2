package app

import (
	"context"
	"fmt"
	"path/filepath"
	"server/api/pb"
	"server/pkg/cfg"
	"server/pkg/db"
	"server/pkg/discovery"
	"server/pkg/flag"
	"server/pkg/ghttp"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/msgq"
	"server/pkg/idgen"
	"server/pkg/lock"
	"server/pkg/logger"
	"server/pkg/thread"
	"server/pkg/util"
	"server/pkg/version"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type App struct {
	Init    func(ctx context.Context) error
	Action  func(ctx context.Context, wait *sync.WaitGroup) error
	UnInit  func(ctx context.Context)
	OnMsg   func(ctx gmsg.Message)
	SrvType pb.Server
}

func (a *App) RootCmdRun(cmd *cobra.Command, args []string) {
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
	if err := idgen.Init(int(flag.SvcIndex)); err != nil {
		return err
	}
	cfg.Load(flag.EtcdAddr[0], flag.IID)
	conf := cfg.Get()

	a.initLog(conf)
	version.LogVersion()

	ghttp.Init(true)

	if err := a.initDB(conf); err != nil {
		return err
	}
	lock.InitPool(db.Redis)
	err := discovery.InitDefault(conf.IID, flag.EtcdAddr, db.Redis)
	if err != nil {
		return err
	}
	discovery.Default.Watch()

	if err := msgq.Q.Init(conf.MsgQueue.SAddr, a.SrvType, flag.SvcIndex, nats.UserInfo(conf.MsgQueue.User, conf.MsgQueue.Pwd)); err != nil {
		return err
	}

	if err := a.Init(ctx); err != nil {
		return err
	}
	// 这之前不能访问mongoDB，因为还未设置dbName
	zap.L().Info("初始化完成")
	return nil
}

func (a *App) initLog(conf *cfg.Config) {
	filePath := filepath.Join("./bin/logs", fmt.Sprintf("%s_%d.log", flag.SrvName(flag.SrvType), flag.SvcIndex))
	logger.NewZapLog(filePath, conf.LogInfo)
}

func (a *App) initDB(conf *cfg.Config) error {
	if err := db.InitRedis(db.RedisCfg{
		Addr:     conf.Redis.Address,
		Password: conf.Redis.Password,
		DB:       conf.Redis.DB,
	}, 0); err != nil {
		return err
	}

	if err := db.InitMongo(conf.Mongo.URL, "admin", 16, 32); err != nil {
		return err
	}
	return nil
}

func (a *App) action(ctx context.Context, wait *sync.WaitGroup) error {
	if err := a.Action(ctx, wait); err != nil {
		return err
	}

	thread.GoSafe(func() {
		ghttp.Serve(ctx, wait, ghttp.EG(), flag.HttpPort)
	})

	if a.OnMsg != nil {
		if err := msgq.Q.Serve(a.OnMsg); err != nil {
			return err
		}
	}
	flag.SetReady()

	if err := discovery.Default.Register(flag.SrvName(a.SrvType), &discovery.Node{NodeID: int32(flag.SvcIndex)}); err != nil {
		return err
	}

	zap.S().Info(util.SuccessShow)
	zap.L().Info("启动成功", zap.String("version", version.GitCommit))

	thread.WaitExit()

	return nil
}

func (a *App) unInit(ctx context.Context) error {
	discovery.Default.Close()
	a.UnInit(ctx)
	// zap.L().Info("closing...")
	_ = db.CloseMongo()
	db.CloseRedis()
	zap.L().Info("服务器关闭")
	return nil
}
