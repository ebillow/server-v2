package logon_service

import (
	"server/pkg/util"
	"sync"

	"go.uber.org/zap"
)

var (
	debugCheck = make(map[uint64]uint64)
	debugMtx   sync.RWMutex
	debugWait  sync.WaitGroup
)

func debugLoginBegin(roleID uint64) {
	if util.Debug {
		debugMtx.Lock()
		debugCheck[roleID] = 0
		debugMtx.Unlock()
		debugWait.Add(1)
	}
}

func debugLoginOk(id uint64) {
	if util.Debug {
		debugMtx.Lock()
		defer debugMtx.Unlock()

		_, ok := debugCheck[id]
		if !ok {
			zap.L().Fatal("id not exists", zap.Uint64("roleID", id))
		} else {
			debugCheck[id] = id
		}

		debugWait.Done()
	}
}

func debugCheckSuccess() bool {
	debugWait.Wait()
	debugMtx.Lock()
	defer debugMtx.Unlock()

	ok := true
	for k, v := range debugCheck {
		if v == 0 {
			ok = false
			zap.L().Error("role login fail", zap.Uint64("role", k))
		}
	}
	return ok
}
