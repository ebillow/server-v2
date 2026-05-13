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

type Registry struct {
	roles map[uint64]meta   // roleID:meta
	ses   map[uint64]uint64 // sesID:roleID
	mtx   sync.RWMutex
}

var Mgr = NewRegistry()

func NewRegistry() *Registry {
	m := &Registry{
		roles: make(map[uint64]meta),
		ses:   make(map[uint64]uint64),
	}

	return m
}

func Run(ctx context.Context) {
	t := time.NewTicker(time.Second * 3)
	t10 := time.NewTicker(time.Second * 10)
	defer func() {
		t.Stop()
		t10.Stop()
	}()
	for {
		select {
		case now := <-t.C:
			Mgr.onTick(now)
		case <-t10.C:
			discovery.UpdateLoad(int32(Mgr.Count()))
		case <-ctx.Done():
			return
		}
	}
}

func (m *Registry) Register(roleID uint64, sesID uint64, r *Role) {
	m.mtx.Lock()
	m.roles[roleID] = meta{
		events: r.Events,
		wait:   &r.Wait,
		cancel: r.Cancel,
		ctx:    r.Ctx,
	}
	m.ses[sesID] = roleID
	m.mtx.Unlock()
}

func (m *Registry) Count() int {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return len(m.roles)
}

func (m *Registry) get(roleID uint64) (meta, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	d, ok := m.roles[roleID]
	return d, ok
}

func (m *Registry) getBySes(sesID uint64) (meta, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if roleID, ok := m.ses[sesID]; !ok {
		return meta{}, false
	} else {
		e, ok := m.roles[roleID]
		return e, ok
	}
}

func (m *Registry) Unregister(roleID uint64, sesID uint64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	// 严谨校验：只有当映射关系匹配时才删除
	if curRoleID, ok := m.ses[sesID]; ok {
		if curRoleID == roleID {
			delete(m.ses, sesID)
			delete(m.roles, roleID)
		}
	}
}

func (m *Registry) onTick(now time.Time) {
	m.mtx.RLock()
	metas := make([]meta, 0, len(m.roles))
	for _, v := range m.roles {
		metas = append(metas, v)
	}
	m.mtx.RUnlock() //  尽早释放读锁
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
	ids := make([]uint64, 0, len(m.roles))
	m.mtx.RLock()
	for id := range m.roles {
		ids = append(ids, id)
	}
	m.mtx.RUnlock()
	for _, id := range ids {
		if r, ok := m.get(id); ok {
			r.Kick() // Signal all immediately
		}
	}
	for _, id := range ids {
		if r, ok := m.get(id); ok {
			r.wait.Wait() // Wait for them to finish concurrently
		}
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
