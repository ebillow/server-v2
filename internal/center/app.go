package center

import (
	"context"
	"server/api/pb"
	"server/internal/center/actors"
	"server/internal/center/onlines"
	"server/internal/share/actor"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gmsg"
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
	actor.Actors.Start()
	return nil
}

func UnInit(ctx context.Context) {
	actor.Actors.StopAndWait()
}

func OnServerMsg(c gmsg.Message) {
	if c.Head.Flag == gmsg.Forward {
		gameID, ok := onlines.GetGameID(c.Head.ActorID)
		if ok {
			err := msgq.Q.Send(pb.Server(c.Head.ToSer), gameID, c.Head.MsgID, c.Data, c.Head.ActorID, c.Head.SesID)
			if err != nil {
				zap.L().Warn("relay err", zap.Error(err), zap.Inline(&c))
			}
		}
	} else if c.Head.ActorID > 0 {
		err := actor.Actors.Dispatch(c.Head.ActorID, actor.Event{Ctx: c})
		if err != nil {
			zap.L().Error("pos msg error", zap.Error(err), zap.Inline(&c))
		}
	} else {
		err := actor.Actors.Dispatch(uint64(pb.ActorID_IDGlobal), actor.Event{Ctx: c})
		if err != nil {
			zap.L().Error("pos msg error", zap.Error(err), zap.Inline(&c))
		}
	}
}
