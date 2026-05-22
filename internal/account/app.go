package account

import (
	"context"
	_ "net/http/pprof"
	"server/api/pb"
	"server/internal/account/acc_db"
	"server/internal/account/auth"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/router"
	"sync"
)

func Init(ctx context.Context) error {
	db.MongoUse(flag.IID + "_account")
	acc_db.CreateIndex()

	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	auth.Start(ctx)
	return nil
}

func UnInit(ctx context.Context) {

}

func OnServerMsg(ctx gctx.Context) {
	if ctx.FromSer == uint8(pb.Server_Gateway) {
		router.C().Handle(ctx)
	} else {
		router.S().Handle(ctx)
	}
}
