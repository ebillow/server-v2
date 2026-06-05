package logon_service

import (
	"server/api/pb"
	"server/internal/game/role"
)

// ILogonEvent 标记接口，所有登录服事件必须实现此接口
type ILogonEvent interface {
	isLogonEvent()
}

// 基础实现，用作组合嵌入，省去每个结构体都写一遍 isLogonEvent
type baseEvent struct{}

func (baseEvent) isLogonEvent() {}

// EvtLogin 登录请求事件
type EvtLogin struct {
	baseEvent
	Login *pb.S2SReqLogin
}

// EvtDBLoaded 数据库加载完毕事件（替代原来的 OpUnmarshal）
type EvtDBLoaded struct {
	baseEvent
	Login *pb.S2SReqLogin
	Data  *role.DataToSave
}

// EvtReentry 顶号重入事件
type EvtReentry struct {
	baseEvent
	Login *pb.S2SReqLogin
}

// EvtLogout 角色下线事件
type EvtLogout struct {
	baseEvent
	Data *role.DataToSave
}

// EvtSaveRole 保存角色事件
type EvtSaveRole struct {
	baseEvent
	Data     *role.DataToSave
	SaveBoth bool
}

// EvtSaveSuccess 保存DB成功事件
type EvtSaveSuccess struct {
	baseEvent
	IDs []uint64
}
