package pub

import (
	"server/api/pb"
	"server/pkg/gnet/dep"
	"sync"
	"sync/atomic"
)

const (
	SvcTypeMax = 64
	SvcIDMax   = 64
)

type Factory[T any] interface {
	NewNodePub(serType pb.Server, serID uint8) *T
	NewAnyPub(serType pb.Server) *T
	NewBroadcastPub(serType pb.Server) *T
}

type Batchers[T any] struct {
	node    [SvcTypeMax][SvcIDMax]atomic.Pointer[T]
	nodeMtx sync.Mutex

	any    [SvcTypeMax]atomic.Pointer[T]
	anyMtx sync.Mutex

	broadcast    [SvcTypeMax]atomic.Pointer[T]
	broadcastMtx sync.Mutex

	factory Factory[T]
}

func (m *Batchers[T]) Init(factory Factory[T]) {
	m.factory = factory
}

func (m *Batchers[T]) Node(serType pb.Server, serID uint8) (*T, error) {
	if serType >= SvcTypeMax || serID >= SvcIDMax {
		return nil, dep.ErrArg
	}

	if tb := m.node[serType][serID].Load(); tb != nil {
		return tb, nil
	}

	m.nodeMtx.Lock()
	defer m.nodeMtx.Unlock()

	if tb := m.node[serType][serID].Load(); tb != nil {
		return tb, nil
	}

	tb := m.factory.NewNodePub(serType, serID)
	m.node[serType][serID].Store(tb)

	return tb, nil
}

func (m *Batchers[T]) Any(serType pb.Server) (*T, error) {
	if serType >= SvcTypeMax {
		return nil, dep.ErrArg
	}

	if tb := m.any[serType].Load(); tb != nil {
		return tb, nil
	}

	m.anyMtx.Lock()
	defer m.anyMtx.Unlock()

	if tb := m.any[serType].Load(); tb != nil {
		return tb, nil
	}

	tb := m.factory.NewAnyPub(serType)
	m.any[serType].Store(tb)

	return tb, nil
}

func (m *Batchers[T]) Broadcast(serType pb.Server) (*T, error) {
	if serType >= SvcTypeMax {
		return nil, dep.ErrArg
	}

	if tb := m.broadcast[serType].Load(); tb != nil {
		return tb, nil
	}

	m.broadcastMtx.Lock()
	defer m.broadcastMtx.Unlock()

	if tb := m.broadcast[serType].Load(); tb != nil {
		return tb, nil
	}

	tb := m.factory.NewBroadcastPub(serType)
	m.broadcast[serType].Store(tb)

	return tb, nil
}

func (m *Batchers[T]) FlushAll() {
	flushFunc := func(tb *T) {
		if tb != nil {
			if f, ok := any(tb).(interface{ Close() }); ok {
				f.Close()
			}
		}
	}

	for i := range m.node {
		for j := range m.node[i] {
			flushFunc(m.node[i][j].Load())
		}
	}
	for i := range m.any {
		flushFunc(m.any[i].Load())
	}
	for i := range m.broadcast {
		flushFunc(m.broadcast[i].Load())
	}
}
