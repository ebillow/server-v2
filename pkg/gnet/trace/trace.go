package trace

import "server/pkg/util"

//
// type ITrace interface {
// 	ShouldLog(msgID uint32, roleID uint64, sessionID uint64) bool
// }

var Rule = NewTraceRule()

type TraceRule struct {
	whiteList map[uint32]struct{}
	blackList map[uint32]struct{}
}

func NewTraceRule() *TraceRule {
	trace := &TraceRule{
		whiteList: make(map[uint32]struct{}),
		blackList: make(map[uint32]struct{}),
	}

	return trace
}

func (t *TraceRule) SetWhiteList(whiteList map[uint32]struct{}) {
	if whiteList != nil {
		return
	}
	t.whiteList = whiteList
}

func (t *TraceRule) SetBlackList(blackList map[uint32]struct{}) {
	if blackList != nil {
		return
	}
	t.blackList = blackList
}

func (t *TraceRule) ShouldLog(msgID uint32, roleID uint64, sessionID uint64) bool {
	if util.Debug { // 除了blackList全追踪
		_, ok := t.blackList[msgID]
		return !ok
	} else {
		_, ok := t.whiteList[msgID]

		// todo 根据roleID和sessionID 决定

		return ok
	}
}
