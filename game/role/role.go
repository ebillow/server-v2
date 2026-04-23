package role

import (
	"context"
	"server/pkg/cfg"
	"server/pkg/flag"
	"server/pkg/gnet"
	"server/pkg/gnet/gctx"
	"server/pkg/model"
	"server/pkg/queue"
	"server/pkg/thread"
	"server/pkg/util"
	"sync"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"

	"server/pkg/pb"
	"time"
)

type DataToSave struct {
	ID   uint64
	Data map[string]string
}

func (d *DataToSave) Get(comID pb.TypeComp) string {
	return d.Data[model.GetCompName(comID)]
}

func (d *DataToSave) Set(comID pb.TypeComp, data string) {
	d.Data[model.GetCompName(comID)] = data
}

func (d *DataToSave) IsEmpty() bool {
	return len(d.Data) == 0
}

const EventChanSize = 16

type Event struct {
	Ctx    gctx.Context
	Func   func(r *Role)
	CliMsg bool
}

// Role	角色数据
type Role struct {
	ID         uint64 // role_mgr需要访问
	SesID      uint64
	Comps      map[pb.TypeComp]IComp
	Data       *pb.RoleData // 入库数据
	CliInfo    *pb.ClientInfo
	ConnectAcc []string
	Seq        uint32

	Events *queue.SwapQueue[Event]
	Wait   sync.WaitGroup
	Ctx    context.Context
	Cancel context.CancelFunc

	// 注意：临时属性，重连后就丢了
	NowSec int64

	dirty            bool
	lastSave         time.Time
	LastMinute       time.Time
	LastHeartbeat    time.Time
	HeartbeatTimeOut int
}

var CreateComps func(r *Role)

// NewRole	新建一个角色
func NewRole(data *DataToSave, login *pb.S2SReqLogin) (*Role, error) {
	dataBase := &pb.RoleData{}

	err := sonic.UnmarshalString(data.Get(pb.TypeComp_TCBase), dataBase)
	if err != nil {
		return nil, err
	}

	r := &Role{
		ID:         data.ID,
		Data:       dataBase,
		SesID:      login.SesID,
		Comps:      make(map[pb.TypeComp]IComp),
		CliInfo:    login.Req.CliInfo,
		ConnectAcc: login.ConnectedAcc,
	}

	// r.Events = make(chan Event, EventChanSize)
	r.Events = queue.NewSwapQueue[Event](EventChanSize, EventChanSize*100)
	r.Ctx, r.Cancel = context.WithCancel(context.Background())

	CreateComps(r)

	for i, comp := range r.Comps {
		compData := data.Get(i)
		if len(compData) == 0 {
			continue
		}
		err = sonic.UnmarshalString(compData, comp)
		if err != nil {
			return nil, err
		}
	}

	if r.Data.CreateTime == 0 {
		r.Data.CreateTime = time.Now().Unix()
		r.SetDirty()
	}

	return r, nil
}

func (r *Role) Loop() {
	r.Wait.Add(1)
	thread.GoSafe(func() {
		defer func() {
			r.Offline()
			r.Wait.Done() // 最后Done
		}()
		r.Online()
		for {
			<-r.Events.Sig()
			r.Events.Range(func(evt Event) bool {
				if evt.Func != nil {
					evt.Func(r)
				} else {
					evt.Ctx.U = r
					if evt.CliMsg {
						cRouter().Handle(evt.Ctx)
					} else {
						sRouter().Handle(evt.Ctx)
					}
				}
				return true
			})
			if r.Ctx.Err() != nil {
				return // 自己退出
			}
		}
	})
}

func (r *Role) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("r.id", r.ID)
	encoder.AddUint64("r.session", r.SesID)
	return nil
}

func (r *Role) Online() {
	now := time.Now()
	r.Data.OnlineTime = now.Unix()
	r.lastSave = now
	r.LastHeartbeat = now
	r.HeartbeatTimeOut = 0

	// 有些数据datareset需要先处理在发给客户端，避免客户端有1s收到头一天的数据
	r.SecLoop(r.lastSave)
	//
	// network.SendToAllCenter(pb.MsgIDS2S_Gm2CtLogin, &pb.MsgKVGuidValue{
	// 	Guid:  r.Data.Guid,
	// 	Value: setup.Setup.ID,
	// })
	//
	for i := range r.Comps {
		if comp, ok := r.Comps[i].(ICompOnline); ok {
			comp.Online(r)
		}
	}

	gnet.SendToGate(&pb.S2SResLogin{
		Res: &pb.S2CLogin{
			Player:     r.Data,
			ConnectAcc: r.ConnectAcc,
		},
		GameID: int32(flag.SvcIndex),
	}, r.SesID)
	zap.L().Info("[login] online", zap.Inline(r))
}

// Offline 角色下线的处理
func (r *Role) Offline() {
	r.Data.OfflineTime = time.Now().Unix()

	for i := range r.Comps {
		if comp, ok := r.Comps[i].(ICompOffline); ok {
			comp.Offline(r)
		}
	}

	RoleMgr().Delete(r.ID, r.SesID)

	data, err := r.marshal(true)
	if err != nil {
		return
	}
	LoginMgr().Offline(data) // offline时在mgr里保存,批量存

	// 通知其它服务器
	r.Disconnect(pb.DisconnectReason_Kick)
	// network.SendToAllCenter(pb.MsgIDS2S_Gm2CtOffline, &pb.MsgKVGuidValue{Guid: r.Data.Guid, Value: setup.Setup.ID})
	zap.L().Info("[login] offline", zap.Inline(r))
}

func (r *Role) Disconnect(why pb.DisconnectReason) {
	gnet.SendToGate(&pb.S2SS2GtDisconnect{
		SesID: r.SesID,
		Why:   why,
	}, r.SesID)
}

func (r *Role) marshal(force bool) (*DataToSave, error) {
	rd := &DataToSave{
		ID:   r.ID,
		Data: make(map[string]string),
	}

	if force || r.dirty {
		str, err := sonic.MarshalString(r.Data)
		if err != nil {
			zap.S().Errorf("marshal role data err:%v", err)
			return nil, err
		}

		rd.Set(pb.TypeComp_TCBase, str)
		r.dirty = false
	}

	for i, v := range r.Comps {
		if !force && !v.IsDirty() {
			continue
		}
		str, err := sonic.MarshalString(v)
		if err != nil {
			zap.L().Error("marshal role comp data err", zap.Error(err), zap.Inline(r))
			continue
		}

		rd.Set(i, str)
		v.ClearDirty()
	}

	return rd, nil
}

func (r *Role) SecLoop(now time.Time) {
	if r.Data == nil {
		zap.L().Error("role.Data == nil")
		return
	}

	r.NowSec = now.Unix()

	reset := false
	dayChange := false
	monthChange := false

	if r.NowSec > r.Data.ResetTime {
		reset = true
		monthChange = r.NowSec >= r.Data.DataResetMonth
	}

	if r.NowSec > r.Data.DayChange {
		dayChange = true
	}

	if now.Sub(r.LastMinute) > time.Minute {
		r.LastMinute = now
		r.MinuteLoop(now)
	}

	for i := range r.Comps {
		if comp, ok := r.Comps[i].(ICompSecLoop); ok {
			comp.SecLoop(now, r)
		}

		if dayChange {
			if comp, ok := r.Comps[i].(ICompDayChange); ok {
				comp.OnDayChange(r)
			}
		}

		if reset { // 每日数据重置
			if comp, ok := r.Comps[i].(ICompDataReset); ok {
				comp.OnDataReset(r)
			}

			if monthChange {
				if comp, ok := r.Comps[i].(ICompMonthChange); ok {
					comp.OnMonthChange(r)
				}
			}
		}
	}

	if reset {
		const ResetHour = 8
		begin := util.CurDayBegin()
		if now.Hour() >= ResetHour {
			r.Data.ResetTime = begin.Add(time.Duration(ResetHour+24) * time.Hour).Unix() // 下一次重置时间
		} else {
			r.Data.ResetTime = begin.Add(time.Duration(ResetHour) * time.Hour).Unix() // 下一次重置时间
		}
		if monthChange {
			curMonthBegin := time.Date(now.Year(), now.Month(), 1, ResetHour, 0, 0, 0, now.Location())
			r.Data.DataResetMonth = curMonthBegin.AddDate(0, 1, 0).Unix()
			zap.S().Debugf("%d data next month reset time=%v", r.ID, time.Unix(r.Data.DataResetMonth, 0))
		}
		// zap.S().Debugf("%d data reset time=%v", r.Guid, time.Unix(r.Data.ResetTime, 0))
	}
	if dayChange {
		begin := util.CurDayBegin()
		r.Data.DayChange = begin.Add(24 * time.Hour).Unix()
		// r.Send(pb.MsgIDS2C_S2CDayChange, nil) // 告知客户端这一天过去了
	}
	if reset {
		// r.Send(pb.MsgIDS2C_S2CDataReset, nil) // 告知客户端数据重置
	}

	conf := cfg.Get()
	if now.Sub(r.lastSave).Seconds() > float64(conf.Time.AutoSave) {
		r.save()
		r.lastSave = now
	}

	const HeartbeatTime = 15 * 2
	if now.Sub(r.LastHeartbeat).Seconds() > float64(HeartbeatTime) {
		r.HeartbeatTimeOut++
		if r.HeartbeatTimeOut > 2 {
			r.Cancel()
		}
	}
}

func (r *Role) MinuteLoop(now time.Time) {
	for i := range r.Comps {
		if iSec, ok := r.Comps[i].(ICompMinuteLoop); ok {
			iSec.MinuteLoop(now, r)
		}
	}
}

// GetComp	获取组件
func (r *Role) GetComp(t pb.TypeComp) IComp {
	return r.Comps[t]
}

// Send	发送数据
func (r *Role) Send(msg proto.Message) {
	gnet.SendToRole(msg, r.SesID, r.ID)
}

func (r *Role) SetDirty() {
	r.dirty = true
}

func (r *Role) save() {
	data, err := r.marshal(false)
	if err != nil {
		return
	}
	if data.IsEmpty() {
		return
	}
	LoginMgr().SaveRole(data) // offline时在mgr里保存,批量存
}
