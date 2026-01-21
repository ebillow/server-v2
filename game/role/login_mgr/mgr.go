package login_mgr

import (
	"context"
	"go.uber.org/zap"
	"server/game/role"
	"server/internal/pb"
	"server/internal/util"
	"sync"
	"time"
)

/* todo
1.json->bson
*/

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
	RoleData  string
	LoginSeq  uint32
}

func (l *loginData) setState(state loginState) {
	l.State = state
	l.StateTime = time.Now().Unix()
}

type Operator struct {
	Op uint32

	Login *pb.C2SLogin   // 上线的参数
	Data  *role.DataInDB // 下线，保存的参数
	IDs   []uint64
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

	util.GoSafe(func() {
		m.waitProducer.Add(1)
		m.run(m.ctx)
	})
	for i := 0; i < LoadingGoCnt; i++ {
		m.waitProducer.Add(1)
		util.GoSafe(func() {
			m.load.run(m.ctx, &m.waitProducer)
		})
	}
	util.GoSafe(func() { // 只能开一个，否则可能后到的先保存
		m.waitConsumer.Add(1)
		m.save.run(&m.waitConsumer)
	})
}

func (m *LoginMgr) Close() {
	m.cancel()

	//	role.GetRoleMgr().WaitAllRoles() //todo

	m.waitProducer.Wait()
	m.waitConsumer.Wait()
}

// Online	请求角色的数据
func (m *LoginMgr) Online(msg *pb.C2SLogin) {
	m.ops <- &Operator{Op: OpOnline, Login: msg}
}

// Offline	角色下线
func (m *LoginMgr) Offline(data *role.DataInDB) {
	m.ops <- &Operator{Op: OpOffline, Data: data}
}

func (m *LoginMgr) SaveRole(data *role.DataInDB) {
	m.ops <- &Operator{Op: OpSaveRole, Data: data}
}

func (m *LoginMgr) postOp(op *Operator) {
	m.ops <- op
}

func postOp(op *Operator) {
	Mgr.postOp(op)
}

func (m *LoginMgr) monitor() {
	zap.L().Info("[login] online monitor", zap.Int("cache count", len(m.data)))
}

func (m *LoginMgr) roleOffline(p *opSaveData) {
	ld, ok := m.data[p.ID]
	if ok {
		ld.setState(stateOffline)
	}
	m.saveOne(p, ld)
}

func (m *LoginMgr) saveOne(p *opSaveData, ld *loginData) {
	if ld != nil {
		ld.RoleData = p.Data
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
			m.saveOne(&opSaveData{ID: k, Data: v.RoleData, Op: OpOffline}, v)
		}
		if v.State == stateCanDel && now-v.StateTime > Interval {
			// todo
			// game_net.SendToRandAccount(&pb.S2SRoleOffline{
			// 	Acc: share.GetAccIDFromRoleID(v.Data.Guid),
			// 	Sn:  v.LoginSn,
			// })
			zap.L().Debug("[login] delete cache", zap.Uint64("id", k))
			delete(m.data, k)
		}
	}
}
