package actor

import (
	"context"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/router"
	"server/pkg/queue"
	"server/pkg/thread"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ISubActor interface {
	Init() error
	OnTick(now time.Time)
	Exit()
}

// Actor 全局Actor

type Event struct {
	Ctx  gmsg.Message
	Func func()
}

type Actor struct {
	Events *queue.SwapQueue[Event]
	sub    ISubActor
}

func NewActor(subActor ISubActor, EventChanSize int) *Actor {
	if EventChanSize == 0 {
		EventChanSize = 40960
	}
	return &Actor{
		sub:    subActor,
		Events: queue.NewSwapQueue[Event](EventChanSize, EventChanSize*100),
	}
}

func (a *Actor) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	thread.GoSafe(func() {
		defer func() {
			a.sub.Exit()
			wg.Done()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				thread.RunSafe(func() {
					a.run(ctx)
				})
			}
		}
	})
}

func (a *Actor) run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer func() {
		t.Stop()
	}()
	for {
		select {
		case <-a.Events.Sig():
			a.Events.Range(func(evt Event) {
				if evt.Func != nil {
					evt.Func()
				} else {
					err := router.R().Handle(evt.Ctx)
					if err != nil {
						zap.L().Warn("actor hand err", zap.Error(err))
					}
				}
			})
		case now := <-t.C:
			a.sub.OnTick(now)
		case <-ctx.Done():
			return
		}
	}
}

func (a *Actor) Post(e Event) error {
	return a.Events.Push(e)
}
