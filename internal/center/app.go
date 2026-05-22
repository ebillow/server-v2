package center

import (
	"context"
	"server/api/pb"
	"server/internal/center/actors"
	"server/internal/center/onlines"
	"server/internal/share/actor"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/msgq"
	"sync"

	"go.uber.org/zap"
)

func Init(ctx context.Context) error {
	db.MongoUse(flag.IID + "_center")
	actors.Create()

	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	actor.Actors.Run()
	return nil
}

func UnInit(ctx context.Context) {
	actor.Actors.StopAndWait()
}

func OnServerMsg(ctx gctx.Context) {
	if ctx.Flag == gctx.Forward {
		gameID, ok := onlines.GetGameID(ctx.ActorID)
		if ok {
			msgq.Q.Send(pb.Server(ctx.ToSer), gameID, ctx.MsgID, ctx.Data, ctx.ActorID, ctx.SesID)
		}
	} else if ctx.ActorID > 0 {
		err := actor.Actors.Post(ctx.ActorID, actor.Event{Ctx: ctx})
		if err != nil {
			zap.L().Error("pos msg error", zap.Error(err))
		}
	} else {
		err := actor.Actors.Post(uint64(pb.ActorID_IDGlobal), actor.Event{Ctx: ctx})
		if err != nil {
			zap.L().Error("pos msg error", zap.Error(err))
		}
	}
}
