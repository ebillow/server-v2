package logon_service

import (
	"context"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/game/role"
	"server/pkg/gnet"
	"server/pkg/thread"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	opChanSize   = 40960
	loadingGoCnt = 3
	saveGoCnt    = 3
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

var Mgr LogonService

type LogonService struct {
	data map[uint64]*loginData // accID:登录数据
	ops  chan ILogonEvent

	load []*loader
	save []*saver

	waitProducer sync.WaitGroup
	waitConsumer sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

func (m *LogonService) Start() {
	m.data = make(map[uint64]*loginData)
	m.ops = make(chan ILogonEvent, opChanSize)
	m.ctx, m.cancel = context.WithCancel(context.Background())

	m.waitProducer.Add(1)
	thread.GoSafe(func() {
		m.run(m.ctx)
	})
	for i := 0; i < loadingGoCnt; i++ {
		m.waitProducer.Add(1)
		l := newLoader(m)
		m.load = append(m.load, l)
		thread.GoSafe(func() {
			l.run(m.ctx, &m.waitProducer)
		})
	}
	for i := 0; i < saveGoCnt; i++ {
		m.waitConsumer.Add(1)
		s := newSaver(m)
		m.save = append(m.save, s)
		thread.GoSafe(func() {
			s.run(&m.waitConsumer)
		})
	}
}

func (m *LogonService) Close() {
	role.Mgr.CloseAndWait()

	m.cancel()

	m.waitProducer.Wait()
	m.waitConsumer.Wait()
}

// Login	请求角色的数据
func (m *LogonService) Login(msg *pb.S2SReqLogin) {
	debugLoginBegin(msg.RoleID)
	m.ops <- &EvtLogin{Login: msg}
}

// Logout	角色下线
func (m *LogonService) Logout(data *role.DataToSave) {
	m.ops <- &EvtLogout{Data: data}
}

func (m *LogonService) SaveRole(data *role.DataToSave, saveBoth bool) bool {
	evt := &EvtSaveRole{Data: data, SaveBoth: saveBoth}
	select {
	case m.ops <- evt:
		return true
	default: // 满了就丢了，Logout会等待存入
		return false
	}
}

func (m *LogonService) postEvent(op ILogonEvent) {
	m.ops <- op
}

func (m *LogonService) postToLoad(op *EvtLogin) {
	idx := op.Login.RoleID % loadingGoCnt
	m.load[idx].loading <- op
}

func (m *LogonService) saveOne(p opSaveData, ld *loginData) {
	if ld != nil {
		ld.Cache = p.Data
	}
	idx := p.ID % saveGoCnt
	m.save[idx].post(p)
}

func (m *LogonService) saveMust(p opSaveData, ld *loginData) {
	if ld != nil {
		ld.Cache = p.Data
	}
	idx := p.ID % saveGoCnt
	m.save[idx].postMustSave(p)
}

func (m *LogonService) postDBLoaded(e *EvtDBLoaded) {
	m.postEvent(e)
}

func (m *LogonService) monitor() {
	zap.L().Info("[login] monitor",
		zap.Int("cache", len(m.data)),
		zap.Int("online", role.Mgr.Count()))
}

func (m *LogonService) roleOffline(p *EvtLogout) {
	ld, ok := m.data[p.Data.RoleID]
	if ok {
		ld.setState(stateOffline)
	}

	m.saveMust(opSaveData{ID: p.Data.RoleID, Data: p.Data.Data, Both: false}, ld)
}

func (m *LogonService) saveSuccess(ids []uint64) {
	for _, id := range ids {
		if v, ok := m.data[id]; ok {
			if v.State == stateOffline {
				v.setState(stateCanDel)
			}
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
			gnet.SendToAccount(msgid.MsgIDS2S_S2SRoleClear, &pb.S2SRoleClear{
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
		// 这里关完了，才关save
		for _, v := range m.save {
			v.close()
		}
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

func (m *LogonService) onOps(_ context.Context, p ILogonEvent) {
	thread.RunSafe(func() {
		switch evt := p.(type) {
		case *EvtLogin:
			m.opSignIn(evt)
		case *EvtDBLoaded:
			m.initRole(evt.Data, evt.Login)
		case *EvtReentry:
			m.opReentry(evt)
		case *EvtLogout:
			m.roleOffline(evt)
		case *EvtSaveRole:
			m.saveOne(opSaveData{ID: evt.Data.RoleID, Data: evt.Data.Data, Both: evt.SaveBoth}, m.data[evt.Data.RoleID])
		case *EvtSaveSuccess:
			m.saveSuccess(evt.IDs)
		default:
			zap.L().Error("[login] unknown event type", zap.Any("event", p))
		}
	})
}

func (m *LogonService) opSignIn(op *EvtLogin) {
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
		m.initRole(&role.DataToSave{RoleID: op.Login.RoleID, Data: v.Cache}, op.Login)
	case statePending:
		now := time.Now()
		if now.Unix()-v.StateTime < StateTimeOut {
			// todo 返回登录失败
			return
		}
		m.postToLoad(op)
		v.setState(statePending)

	case stateKicking:
		return
	default:
		v.setState(statePending)
		m.postToLoad(op)
	}
}

func (m *LogonService) initRole(data *role.DataToSave, login *pb.S2SReqLogin) {
	r, err := role.NewRole(data, login)
	if err != nil {
		zap.S().Errorf("new role err:%v", err)
		if v, ok := m.data[data.RoleID]; ok {
			v.setState(stateOffline) // 退回到离线状态
		}
		return
	}

	if r.ID != login.RoleID {
		if v, ok := m.data[data.RoleID]; ok {
			v.setState(stateOffline) // 退回到离线状态
		}
		zap.L().Error("role id and login id are not the same", zap.Uint64("role_id", r.ID), zap.Uint64("login_id", login.RoleID))
		return
	}

	v := m.data[r.ID]
	v.Cache = data.Data
	v.LoginSeq = login.Seq
	v.setState(stateOnline)
	role.Mgr.Register(r.ID, r.SesID, r)

	r.Run()

	debugLoginOk(r.ID)
}

// 处理其它设备
func (m *LogonService) handleReentry(v *loginData, p *EvtLogin) {
	// 避免role协程已退出了，不在role协程处理，
	// 避免阻塞login协程，不在login协程wait
	v.setState(stateKicking)

	thread.GoSafe(func() { // 这里角色数据做参数的话，offline里就不能修改数据了
		role.Mgr.KickAndWait(p.Login.RoleID) // 可以wait多次

		m.postEvent(&EvtReentry{
			Login: p.Login,
		})
		zap.L().Debug("[login] onLoginRepeated", zap.Uint64("id", p.Login.RoleID))
	})
}

func (m *LogonService) opReentry(p *EvtReentry) {
	v := m.data[p.Login.RoleID]
	if v == nil {
		zap.L().Warn("[login] can not find login data")
		return
	}

	zap.L().Debug("[login] opLoginRepeated", zap.Uint64("id", p.Login.RoleID), zap.Any("data", v.Cache))
	m.initRole(&role.DataToSave{RoleID: p.Login.RoleID, Data: v.Cache}, p.Login)
}
