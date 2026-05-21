package role

import (
	"context"
	"server/pkg/discovery"
	"server/pkg/gerror"
	"server/pkg/queue"
	"sync"
	"time"

	"go.uber.org/zap"
)

const shardCount = 64
const tickTime = 3

type meta struct {
	events *queue.SwapQueue[Event]
	wait   *sync.WaitGroup
	cancel context.CancelFunc
	ctx    context.Context
}

func (m meta) Kick() {
	m.cancel()
	m.events.Wake()
}

// 定義獨立的角色分片體系
type roleShard struct {
	mtx   sync.RWMutex
	roles map[uint64]meta
}

// 定義獨立的會話分片體系
type sesShard struct {
	mtx sync.RWMutex
	ses map[uint64]uint64
}

type Registry struct {
	roleShards [shardCount]*roleShard
	sesShards  [shardCount]*sesShard
}

var Mgr = NewRegistry()

func NewRegistry() *Registry {
	m := &Registry{}
	// 初始化所有分片
	for i := 0; i < shardCount; i++ {
		m.roleShards[i] = &roleShard{roles: make(map[uint64]meta)}
		m.sesShards[i] = &sesShard{ses: make(map[uint64]uint64)}
	}
	return m
}

func Run(ctx context.Context) {
	const totalTickCycle = tickTime * time.Second
	tickerDuration := totalTickCycle / time.Duration(shardCount)

	t := time.NewTicker(tickerDuration)
	tUpLoad := time.NewTicker(time.Second * 3)
	defer func() {
		t.Stop()
		tUpLoad.Stop()
	}()

	curTick := 0

	for {
		select {
		case now := <-t.C:
			Mgr.onTick(now, curTick)
			curTick++
			if curTick >= shardCount {
				curTick = 0
			}
		case <-tUpLoad.C:
			discovery.UpdateLoad(int32(Mgr.Count()))
		case <-ctx.Done():
			return
		}
	}
}

func (m *Registry) getRoleShard(roleID uint64) *roleShard {
	return m.roleShards[roleID&(shardCount-1)]
}

func (m *Registry) getSesShard(sesID uint64) *sesShard {
	return m.sesShards[sesID&(shardCount-1)]
}

func (m *Registry) Register(roleID uint64, sesID uint64, r *Role) {
	sShard := m.getSesShard(sesID)
	sShard.mtx.Lock()
	sShard.ses[sesID] = roleID
	sShard.mtx.Unlock()

	rShard := m.getRoleShard(roleID)
	rShard.mtx.Lock()
	rShard.roles[roleID] = meta{
		events: r.Events,
		wait:   &r.Wait,
		cancel: r.Cancel,
		ctx:    r.Ctx,
	}
	rShard.mtx.Unlock()
}

func (m *Registry) Count() int {
	var count int
	for _, shard := range m.roleShards {
		shard.mtx.RLock()
		count += len(shard.roles)
		shard.mtx.RUnlock()
	}
	return count
}

func (m *Registry) get(roleID uint64) (meta, bool) {
	rs := m.getRoleShard(roleID)
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()
	d, ok := rs.roles[roleID]
	return d, ok
}

func (m *Registry) getBySes(sesID uint64) (meta, bool) {
	ss := m.getSesShard(sesID)
	ss.mtx.RLock()
	roleID, ok := ss.ses[sesID]
	ss.mtx.RUnlock()

	if !ok {
		return meta{}, false
	}
	return m.get(roleID)
}

func (m *Registry) Unregister(roleID uint64, sesID uint64) {
	ss := m.getSesShard(sesID)

	ss.mtx.Lock()
	curRoleID, ok := ss.ses[sesID]
	if ok && curRoleID == roleID {
		delete(ss.ses, sesID)
	}
	ss.mtx.Unlock()

	if ok && curRoleID == roleID {
		rs := m.getRoleShard(roleID)
		rs.mtx.Lock()
		delete(rs.roles, roleID)
		rs.mtx.Unlock()
	}
}

func (m *Registry) onTick(now time.Time, shardIdx int) {
	var metas []meta
	shard := m.roleShards[shardIdx]
	shard.mtx.RLock()
	for _, v := range shard.roles {
		metas = append(metas, v)
	}
	shard.mtx.RUnlock() // 尽早释放当前分片的读锁

	for _, v := range metas {
		err := v.events.PushAndWake(Event{Func: func(r *Role) {
			r.OnTick(now)
		}})
		if err != nil {
			zap.L().Error("role secLoop err", zap.Error(err))
		}
	}
}

func (m *Registry) KickAndWait(roleID uint64) {
	r, ok := m.get(roleID)
	if !ok {
		return
	}
	r.Kick()
	r.wait.Wait()
}

func (m *Registry) Kick(sesID uint64) {
	r, ok := m.getBySes(sesID)
	if !ok {
		return
	}
	r.Kick()
}

func (m *Registry) CloseAndWait() {
	var metas []meta
	for _, shard := range m.roleShards {
		shard.mtx.RLock()
		if len(shard.roles) > 0 {
			for _, v := range shard.roles {
				metas = append(metas, v)
			}
		}
		shard.mtx.RUnlock()
	}

	for _, v := range metas {
		v.Kick()
	}
	for _, v := range metas {
		v.wait.Wait()
	}
}

func (m *Registry) Dispatch(roleID uint64, evt Event) error {
	r, ok := m.get(roleID)
	if !ok {
		return gerror.New("role not exist")
	}
	if err := r.events.PushAndWake(evt); err != nil {
		return gerror.New("role event full")
	}
	return nil
}

func (m *Registry) DispatchBySesID(sesID uint64, evt Event) error {
	r, ok := m.getBySes(sesID)
	if !ok {
		return gerror.New("role not exist")
	}
	if err := r.events.PushAndWake(evt); err != nil {
		return gerror.New("role event full")
	}
	return nil
}
