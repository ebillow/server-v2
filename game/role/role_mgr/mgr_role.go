package role_mgr

import (
	"context"
	"go.uber.org/zap"
	"server/game/role"
	"sync"
)

type meta struct {
	events chan role.Event
	wait   *sync.WaitGroup
	cancel context.CancelFunc
}

type RoleMgr struct {
	roles map[uint64]meta   // roleID:meta
	ses   map[uint64]uint64 // sesID:roleID
	mtx   sync.RWMutex
}

var Mgr = NewRoleMgr()

func NewRoleMgr() *RoleMgr {
	return &RoleMgr{
		roles: make(map[uint64]meta),
		ses:   make(map[uint64]uint64),
	}
}

func (m *RoleMgr) Add(roleID uint64, sesID uint64, r *role.Role) {
	m.mtx.Lock()
	m.roles[roleID] = meta{
		events: r.Events,
		wait:   &r.Wait,
		cancel: r.Cancel,
	}
	m.ses[sesID] = roleID
	m.mtx.Unlock()
}

func (m *RoleMgr) Count() int {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return len(m.roles)
}

func (m *RoleMgr) get(roleID uint64) (meta, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	d, ok := m.roles[roleID]
	return d, ok
}

func (m *RoleMgr) getBySes(sesID uint64) (meta, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if roleID, ok := m.ses[sesID]; !ok {
		return meta{}, false
	} else {
		e, ok := m.roles[roleID]
		return e, ok
	}
}

func (m *RoleMgr) Delete(roleID uint64, sesID uint64) {
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

func (m *RoleMgr) KickRoleAndWait(roleID uint64) {
	r, ok := m.get(roleID)
	if !ok {
		return
	}
	r.cancel()
	r.wait.Wait()
}

func (m *RoleMgr) Kick(sesID uint64) {
	r, ok := m.getBySes(sesID)
	if !ok {
		return
	}
	r.cancel()
}

func (m *RoleMgr) CloseAndWait() {
	ids := make([]uint64, 0, len(m.roles))
	m.mtx.RLock()
	for id := range m.roles {
		ids = append(ids, id)
	}
	m.mtx.RUnlock()
	for _, id := range ids {
		m.KickRoleAndWait(id)
	}
}

func (m *RoleMgr) PostEvent(roleID uint64, evt role.Event) {
	r, ok := m.get(roleID)
	if !ok {
		return
	}
	select {
	case r.events <- evt:
	default:
		zap.L().Warn("role_mgr.postEvent chan full", zap.Uint64("roleId", roleID))
	}
}

func (m *RoleMgr) PostEventBySesID(sesID uint64, evt role.Event) {
	r, ok := m.getBySes(sesID)
	if !ok {
		return
	}
	select {
	case r.events <- evt:
	default:
		zap.L().Warn("role_mgr.postEvent chan full", zap.Uint64("roleId", sesID))
	}
}
