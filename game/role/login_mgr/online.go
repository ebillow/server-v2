package login_mgr

import (
	"context"
	"go.uber.org/zap"
	"server/game/role"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/thread"
	"time"
)

func (m *LoginMgr) run(ctx context.Context) {
	tMinute := time.NewTicker(time.Minute)
	defer func() {
		tMinute.Stop()
		m.save.close()
		m.waitProducer.Done()
	}()
	for {
		select {
		case p := <-m.ops:
			m.onOps(ctx, p)
		case <-tMinute.C:
			m.checkClear()
			m.monitor()
		case <-m.ctx.Done():
			m.drainOps()
			return
		}
	}
}

func (m *LoginMgr) drainOps() {
	for {
		select {
		case p := <-m.ops:
			m.onOps(context.Background(), p)
		default:
			return
		}
	}
}

func (m *LoginMgr) onOps(ctx context.Context, p *Operator) {
	thread.RunSafe(func() {
		switch p.Op {
		case OpOnline:
			m.opOnline(ctx, p)
		case OpUnmarshal:
			m.unmarshal(ctx, p.Data, p.Login)
		case OpRepeatedLogin:
			m.opLoginRepeated(ctx, p)
		case OpOffline:
			m.roleOffline(&opSaveData{ID: p.Data.ID, Data: p.Data.Data, Op: OpOffline})
		case OpSaveRole:
			m.saveOne(&opSaveData{ID: p.Data.ID, Data: p.Data.Data, Op: OpSaveRole}, m.data[p.Data.ID])
		case OpSaveSuccess:
			m.saveSuccess(p.IDs)
		}
	})
}

func (m *LoginMgr) opOnline(ctx context.Context, op *Operator) {
	zap.L().Debug("[login] opOnline", zap.Uint64("id", op.Login.RoleID))
	const StateTimeOut = 10
	v := m.data[op.Login.RoleID]
	if v == nil {
		v = &loginData{State: stateInit}
		m.data[op.Login.RoleID] = v
	}
	switch v.State {
	case stateOnline: // 重复登录
		m.onLoginRepeated(v, op)
	case stateOffline, stateCanDel:
		m.unmarshal(ctx, &role.DataToSave{ID: op.Login.RoleID, Data: v.Cache}, op.Login)
	case statePending:
		now := time.Now()
		if now.Unix()-v.StateTime < StateTimeOut {
			return
		} else {
			m.load.post(op)
			v.setState(statePending)
		}
	case stateKicking:
		return
	default:
		v.setState(statePending)
		m.load.post(op)
	}
}

func (m *LoginMgr) unmarshal(ctx context.Context, data *role.DataToSave, login *pb.S2SReqLogin) {
	r, err := role.NewRole(data, login)
	if err != nil {
		logger.Errorf("new role err:%v", err)
		return
	}

	v := m.data[r.ID]
	v.Cache = data.Data
	v.LoginSeq = login.Seq
	v.setState(stateOnline)
	role.RoleMgr().Add(r.ID, r.SesID, r)

	r.Loop(ctx)
}

// 处理其它设备
func (m *LoginMgr) onLoginRepeated(v *loginData, p *Operator) {
	// 避免role协程已退出了，不在role协程处理，
	// 避免阻塞login协程，不在login协程wait
	v.setState(stateKicking)

	thread.GoSafe(func() { // 这里带数据的话，offline里就不能修改数据了
		role.RoleMgr().PostCloseAndWait(p.Login.RoleID) // 可以wait多次
		p.Op = OpRepeatedLogin
		m.ops <- p
		zap.L().Debug("[login] onLoginRepeated", zap.Uint64("id", p.Login.RoleID))
	})
}

func (m *LoginMgr) opLoginRepeated(ctx context.Context, p *Operator) {
	v := m.data[p.Login.RoleID]
	if v == nil {
		zap.L().Warn("[login] can not find login data")
		return
	}

	zap.L().Debug("[login] opLoginRepeated", zap.Uint64("id", p.Login.RoleID), zap.Any("data", v.Cache))
	m.unmarshal(ctx, &role.DataToSave{ID: p.Login.RoleID, Data: v.Cache}, p.Login)
}
