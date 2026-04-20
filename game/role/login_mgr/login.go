package login_mgr

import (
	"context"
	"server/game/role"
	"server/game/role/role_mgr"
	"server/pkg/gnet"
	"server/pkg/pb"
	"server/pkg/thread"
	"server/pkg/util"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	OpChanSize   = 40960
	LoadingGoCnt = 3
)

const (
	OpOnline uint32 = iota
	OpUnmarshal
	OpRepeatedLogin
	OpOffline
	OpSaveRole
	OpSaveSuccess
)

type loginState int

const (
	stateInit loginState = iota
	stateOnline
	statePending
	stateKicking
	stateOffline
	stateCanDel
)

type loginData struct {
	State     loginState
	StateTime int64
	Cache     map[string]string
	LoginSeq  uint32
}

func (l *loginData) setState(state loginState) {
	l.State = state
	l.StateTime = time.Now().Unix()
}

type Operator struct {
	Op uint32

	Login *pb.S2SReqLogin  // 上线的参数
	Data  *role.DataToSave // 下线，保存的参数
	IDs   []uint64
}

var (
	debugCheck = make(map[uint64]uint64)
	debugMtx   sync.RWMutex
	debugWait  sync.WaitGroup
)

func DebugLoginOk(id uint64) {
	if util.Debug {
		debugMtx.Lock()
		debugCheck[id] = id
		debugMtx.Unlock()
		debugWait.Done()
	}
}

var Mgr LoginMgr

type LoginMgr struct {
	data map[uint64]*loginData // accID:登录数据
	ops  chan *Operator

	load *loader
	save *saver

	waitProducer sync.WaitGroup
	waitConsumer sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

func (m *LoginMgr) Start() {
	m.data = make(map[uint64]*loginData)
	m.ops = make(chan *Operator, OpChanSize)
	m.load = newLoader()
	m.save = newSaver()

	m.ctx, m.cancel = context.WithCancel(context.Background())

	thread.GoSafe(func() {
		m.waitProducer.Add(1)
		m.run(m.ctx)
	})
	for i := 0; i < LoadingGoCnt; i++ {
		m.waitProducer.Add(1)
		thread.GoSafe(func() {
			m.load.run(m.ctx, &m.waitProducer)
		})
	}
	thread.GoSafe(func() { // 只能开一个，否则可能后到的先保存
		m.waitConsumer.Add(1)
		m.save.run(&m.waitConsumer)
	})
}

func (m *LoginMgr) Close() {
	m.cancel()

	role.RoleMgr().CloseAndWait()

	m.waitProducer.Wait()
	m.waitConsumer.Wait()
}

// Online	请求角色的数据
func (m *LoginMgr) Online(msg *pb.S2SReqLogin) {
	if util.Debug {
		debugMtx.Lock()
		debugCheck[msg.RoleID] = 0
		debugMtx.Unlock()
		debugWait.Add(1)
	}

	m.ops <- &Operator{Op: OpOnline, Login: msg}
}

// Offline	角色下线
func (m *LoginMgr) Offline(data *role.DataToSave) {
	m.ops <- &Operator{Op: OpOffline, Data: data}
}

func (m *LoginMgr) SaveRole(data *role.DataToSave) {
	m.ops <- &Operator{Op: OpSaveRole, Data: data}
}

func (m *LoginMgr) postOp(op *Operator) {
	m.ops <- op
}

func postOp(op *Operator) {
	Mgr.postOp(op)
}

func (m *LoginMgr) monitor() {
	zap.L().Info("[login] monitor",
		zap.Int("cache", len(m.data)),
		zap.Int("online", role_mgr.Mgr.Count()))
}

func (m *LoginMgr) roleOffline(p opSaveData) {
	ld, ok := m.data[p.ID]
	if ok {
		ld.setState(stateOffline)
	}
	m.saveOne(p, ld)
}

func (m *LoginMgr) saveOne(p opSaveData, ld *loginData) {
	if ld != nil {
		ld.Cache = p.Data
	}
	m.save.post(p)
}

func (m *LoginMgr) saveSuccess(ids []uint64) {
	for _, id := range ids {
		if v, ok := m.data[id]; ok {
			v.setState(stateCanDel)
		}
	}
}

func (m *LoginMgr) checkClear() {
	now := time.Now().Unix()
	const Interval = int64(60 * 1)

	for k, v := range m.data {
		if v.State == stateOffline && now-v.StateTime > Interval {
			m.saveOne(opSaveData{ID: k, Data: v.Cache, Op: OpOffline}, v)
		}
		if v.State == stateCanDel && now-v.StateTime > Interval {
			gnet.SendToAccount(&pb.S2SRoleClear{
				RoleID: k,
				Seq:    v.LoginSeq,
			})
			zap.L().Debug("[login] delete cache", zap.Uint64("id", k))
			delete(m.data, k)
		}
	}
}

func (m *LoginMgr) run(ctx context.Context) {
	tMinute := time.NewTicker(time.Minute)
	defer func() {
		tMinute.Stop()
		m.save.close() // 这里关完了，才关save
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
			m.roleOffline(opSaveData{ID: p.Data.ID, Data: p.Data.Data, Op: OpOffline})
		case OpSaveRole:
			m.saveOne(opSaveData{ID: p.Data.ID, Data: p.Data.Data, Op: OpSaveRole}, m.data[p.Data.ID])
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
		zap.S().Errorf("new role err:%v", err)
		return
	}

	if r.ID != login.RoleID {
		zap.L().Error("role id and login id are not the same", zap.Uint64("role_id", r.ID), zap.Uint64("login_id", login.RoleID))
		return
	}

	v := m.data[r.ID]
	v.Cache = data.Data
	v.LoginSeq = login.Seq
	v.setState(stateOnline)
	role.RoleMgr().Add(r.ID, r.SesID, r)

	r.Loop(ctx)

	DebugLoginOk(r.ID)
}

// 处理其它设备
func (m *LoginMgr) onLoginRepeated(v *loginData, p *Operator) {
	// 避免role协程已退出了，不在role协程处理，
	// 避免阻塞login协程，不在login协程wait
	v.setState(stateKicking)

	thread.GoSafe(func() { // 这里角色数据做参数的话，offline里就不能修改数据了
		role.RoleMgr().KickRoleAndWait(p.Login.RoleID) // 可以wait多次
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
