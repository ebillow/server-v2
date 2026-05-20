package logon_service

import (
	"context"
	"server/game/role"
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
	OpLogin uint32 = iota
	OpUnmarshal
	OpReentry
	OpLogout
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
	IDs   []uint64
	Login *pb.S2SReqLogin  // 上线的参数
	Data  *role.DataToSave // 下线，保存的参数

	Op       uint32
	SaveBoth bool
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

var Mgr LogonService

type LogonService struct {
	data map[uint64]*loginData // accID:登录数据
	ops  chan *Operator

	load *loader
	save *saver

	waitProducer sync.WaitGroup
	waitConsumer sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

func (m *LogonService) Start() {
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

func (m *LogonService) Close() {
	m.cancel()

	role.Mgr.CloseAndWait()

	m.waitProducer.Wait()
	m.waitConsumer.Wait()
}

// Login	请求角色的数据
func (m *LogonService) Login(msg *pb.S2SReqLogin) {
	if util.Debug {
		debugMtx.Lock()
		debugCheck[msg.RoleID] = 0
		debugMtx.Unlock()
		debugWait.Add(1)
	}

	m.ops <- &Operator{Op: OpLogin, Login: msg}
}

// Logout	角色下线
func (m *LogonService) Logout(data *role.DataToSave) {
	m.ops <- &Operator{Op: OpLogout, Data: data}
}

func (m *LogonService) SaveRole(data *role.DataToSave, saveBoth bool) {
	m.ops <- &Operator{Op: OpSaveRole, Data: data, SaveBoth: saveBoth}
}

func (m *LogonService) postOp(op *Operator) {
	m.ops <- op
}

func postOp(op *Operator) {
	Mgr.postOp(op)
}

func (m *LogonService) monitor() {
	zap.L().Info("[login] monitor",
		zap.Int("cache", len(m.data)),
		zap.Int("online", role.Mgr.Count()))
}

func (m *LogonService) roleOffline(p *Operator) {
	ld, ok := m.data[p.Data.ID]
	if ok {
		ld.setState(stateOffline)
	}

	m.saveOne(opSaveData{ID: p.Data.ID, Data: p.Data.Data, Both: p.SaveBoth}, ld)
}

func (m *LogonService) saveOne(p opSaveData, ld *loginData) {
	if ld != nil {
		ld.Cache = p.Data
	}
	m.save.post(p)
}

func (m *LogonService) saveSuccess(ids []uint64) {
	for _, id := range ids {
		if v, ok := m.data[id]; ok {
			v.setState(stateCanDel)
		}
	}
}

func (m *LogonService) cleanup() {
	now := time.Now().Unix()
	const Interval = int64(60 * 1)

	for k, v := range m.data {
		if v.State == stateOffline && now-v.StateTime > Interval {
			m.saveOne(opSaveData{ID: k, Data: v.Cache, Both: true}, v)
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

func (m *LogonService) run(ctx context.Context) {
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
			m.cleanup()
			m.monitor()
		case <-m.ctx.Done():
			m.drainOps()
			return
		}
	}
}

func (m *LogonService) drainOps() {
	for {
		select {
		case p := <-m.ops:
			m.onOps(context.Background(), p)
		default:
			return
		}
	}
}

func (m *LogonService) onOps(ctx context.Context, p *Operator) {
	thread.RunSafe(func() {
		switch p.Op {
		case OpLogin:
			m.opSignIn(p)
		case OpUnmarshal:
			m.initRole(p.Data, p.Login)
		case OpReentry:
			m.opReentry(p)
		case OpLogout:
			m.roleOffline(p)
		case OpSaveRole:
			m.saveOne(opSaveData{ID: p.Data.ID, Data: p.Data.Data, Both: p.SaveBoth}, m.data[p.Data.ID])
		case OpSaveSuccess:
			m.saveSuccess(p.IDs)
		}
	})
}

func (m *LogonService) opSignIn(op *Operator) {
	const StateTimeOut = 10
	v := m.data[op.Login.RoleID]
	if v == nil {
		v = &loginData{State: stateInit}
		m.data[op.Login.RoleID] = v
	}
	switch v.State {
	case stateOnline: // 重复登录
		m.handleReentry(v, op)
	case stateOffline, stateCanDel:
		m.initRole(&role.DataToSave{ID: op.Login.RoleID, Data: v.Cache}, op.Login)
	case statePending:
		now := time.Now()
		if now.Unix()-v.StateTime < StateTimeOut {
			return
		}
		m.load.post(op)
		v.setState(statePending)

	case stateKicking:
		return
	default:
		v.setState(statePending)
		m.load.post(op)
	}
}

func (m *LogonService) initRole(data *role.DataToSave, login *pb.S2SReqLogin) {
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
	role.Mgr.Register(r.ID, r.SesID, r)

	r.Run()

	DebugLoginOk(r.ID)
}

// 处理其它设备
func (m *LogonService) handleReentry(v *loginData, p *Operator) {
	// 避免role协程已退出了，不在role协程处理，
	// 避免阻塞login协程，不在login协程wait
	v.setState(stateKicking)

	thread.GoSafe(func() { // 这里角色数据做参数的话，offline里就不能修改数据了
		role.Mgr.KickAndWait(p.Login.RoleID) // 可以wait多次
		p.Op = OpReentry
		m.ops <- p
		zap.L().Debug("[login] onLoginRepeated", zap.Uint64("id", p.Login.RoleID))
	})
}

func (m *LogonService) opReentry(p *Operator) {
	v := m.data[p.Login.RoleID]
	if v == nil {
		zap.L().Warn("[login] can not find login data")
		return
	}

	zap.L().Debug("[login] opLoginRepeated", zap.Uint64("id", p.Login.RoleID), zap.Any("data", v.Cache))
	m.initRole(&role.DataToSave{ID: p.Login.RoleID, Data: v.Cache}, p.Login)
}
