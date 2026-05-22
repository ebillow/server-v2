package actor

import (
	"context"
	"fmt"
	"server/pkg/gerror"
	"sync"

	"go.uber.org/zap"
)

var Actors = NewActorMgr()

// ActorMgr 全局Actor管理，目前没动态增减需求
type ActorMgr struct {
	actors map[uint64]*Actor
	wg     sync.WaitGroup
	cancel context.CancelFunc
	ctx    context.Context
}

func NewActorMgr() *ActorMgr {
	ctx, cancel := context.WithCancel(context.Background())
	return &ActorMgr{
		actors: make(map[uint64]*Actor),
		cancel: cancel,
		ctx:    ctx,
	}
}

func (m *ActorMgr) Init(id uint64, a ISubActor, evtChanSize int) error {
	if _, ok := m.actors[id]; ok {
		return gerror.New(fmt.Sprintf("actor id %d repeated", id))
	}
	ac := NewActor(a, evtChanSize)
	ac.sub.Init()

	m.actors[id] = ac
	return nil
}

func (m *ActorMgr) Run() {
	for _, v := range m.actors {
		v.Start(m.ctx, &m.wg)
	}
}

func (m *ActorMgr) StopAndWait() {
	m.cancel()
	for k, v := range m.actors {
		_ = v.Post(Event{Func: func() {
			zap.L().Info("actor stopped", zap.Uint64("id", k))
		}})
	}
	m.wg.Wait()
}

func (m *ActorMgr) get(id uint64) (*Actor, bool) {
	a, ok := m.actors[id]
	return a, ok
}

func (m *ActorMgr) Post(id uint64, e Event) error {
	a, ok := m.get(id)
	if !ok {
		return gerror.New(fmt.Sprintf("actor id %d not found", id))
	}
	return a.Post(e)
}
