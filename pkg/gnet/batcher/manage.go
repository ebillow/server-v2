package batcher

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
	NewIdx(serType pb.Server, serID uint8) *T
	NewGroup(serType pb.Server) *T
	NewAll(serType pb.Server) *T
}

type BatcherManager[T any] struct {
	pubIDXs   [SvcTypeMax][SvcIDMax]atomic.Pointer[T]
	pubIDXMtx sync.Mutex

	pubGroup    [SvcTypeMax]atomic.Pointer[T]
	pubGroupMtx sync.Mutex

	pubAll    [SvcTypeMax]atomic.Pointer[T]
	pubAllMtx sync.Mutex

	factory Factory[T]
}

func (m *BatcherManager[T]) Init(factory Factory[T]) {
	m.factory = factory
}

func (m *BatcherManager[T]) GetIdx(serType pb.Server, serID uint8) (*T, error) {
	if serType >= SvcTypeMax || serID >= SvcIDMax {
		return nil, dep.ErrArg
	}

	if tb := m.pubIDXs[serType][serID].Load(); tb != nil {
		return tb, nil
	}

	m.pubIDXMtx.Lock()
	defer m.pubIDXMtx.Unlock()

	if tb := m.pubIDXs[serType][serID].Load(); tb != nil {
		return tb, nil
	}

	tb := m.factory.NewIdx(serType, serID)
	m.pubIDXs[serType][serID].Store(tb)

	return tb, nil
}

func (m *BatcherManager[T]) GetGroup(serType pb.Server) (*T, error) {
	if serType >= SvcTypeMax {
		return nil, dep.ErrArg
	}

	if tb := m.pubGroup[serType].Load(); tb != nil {
		return tb, nil
	}

	m.pubGroupMtx.Lock()
	defer m.pubGroupMtx.Unlock()

	if tb := m.pubGroup[serType].Load(); tb != nil {
		return tb, nil
	}

	tb := m.factory.NewGroup(serType)
	m.pubGroup[serType].Store(tb)

	return tb, nil
}

func (m *BatcherManager[T]) GetAll(serType pb.Server) (*T, error) {
	if serType >= SvcTypeMax {
		return nil, dep.ErrArg
	}

	if tb := m.pubAll[serType].Load(); tb != nil {
		return tb, nil
	}

	m.pubAllMtx.Lock()
	defer m.pubAllMtx.Unlock()

	if tb := m.pubAll[serType].Load(); tb != nil {
		return tb, nil
	}

	tb := m.factory.NewAll(serType)
	m.pubAll[serType].Store(tb)

	return tb, nil
}

func (m *BatcherManager[T]) FlushAll() {
	flushFunc := func(tb *T) {
		if tb != nil {
			if f, ok := any(tb).(interface{ StopAndFlush() }); ok {
				f.StopAndFlush()
			}
		}
	}

	for i := range m.pubIDXs {
		for j := range m.pubIDXs[i] {
			flushFunc(m.pubIDXs[i][j].Load())
		}
	}
	for i := range m.pubGroup {
		flushFunc(m.pubGroup[i].Load())
	}
	for i := range m.pubAll {
		flushFunc(m.pubAll[i].Load())
	}
}
