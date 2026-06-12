package account

import (
	"context"
	_ "net/http/pprof"
	"server/internal/account/acc_db"
	"server/internal/account/auth"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/router"
	"sync"

	"go.uber.org/zap"
)

func Init(ctx context.Context) error {
	db.MongoUse(flag.IID + "_account")
	acc_db.CreateIndex()

	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	auth.StartService(ctx)
	return nil
}

func UnInit(ctx context.Context) {

}

func OnServerMsg(c gmsg.Message) {
	err := router.R().Handle(c)
	if err != nil {
		zap.L().Info("<<< msg.recv:",
			zap.Inline(&c),
		)
	}
}
