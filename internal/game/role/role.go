package role

import (
	"context"
	"fmt"
	"server/api/pb"
	"server/api/pb/msgid"
	"server/internal/share/model"
	"server/pkg/cfg"
	"server/pkg/flag"
	"server/pkg/gnet"
	"server/pkg/gnet/pkg"
	"server/pkg/gnet/router"
	"server/pkg/queue"
	"server/pkg/thread"
	"server/pkg/util"
	"sync"

	"time"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type DataToSave struct {
	RoleID uint64
	Data   map[string]string
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
	Ctx  pkg.Packet
	Func func(r *Role)
}

// Role	角色数据
type Role struct {
	ID         uint64 // role_mgr需要访问
	SesID      uint64
	Comps      []IComp
	Data       *pb.RoleData // 入库数据
	CliInfo    *pb.ClientInfo
	ConnectAcc []string
	Seq        uint32

	Events *queue.SwapQueue[Event]
	Wait   sync.WaitGroup
	Ctx    context.Context
	Cancel context.CancelFunc

	// 注意：临时属性，重连后就丢了
	nowSec           int64 // 当前时间，精确到秒
	lastSaveRedis    time.Time
	lastSaveDB       time.Time
	lastMinute       time.Time
	LastHeartbeat    time.Time
	HeartbeatTimeOut int
	dirty            bool
}

// NewRole	新建一个角色
func NewRole(data *DataToSave, login *pb.S2SReqLogin) (*Role, error) {
	dataBase := &pb.RoleData{}

	err := sonic.UnmarshalString(data.Get(pb.TypeComp_TCBase), dataBase)
	if err != nil {
		return nil, err
	}

	r := &Role{
		ID:         data.RoleID,
		Data:       dataBase,
		SesID:      login.SesID,
		Comps:      make([]IComp, pb.TypeComp_TCMax),
		CliInfo:    login.Req.CliInfo,
		ConnectAcc: login.ConnectedAcc,
	}

	r.Events = queue.NewSwapQueue[Event](EventChanSize, EventChanSize*100)
	r.Ctx, r.Cancel = context.WithCancel(context.Background())

	compCreate.Create(r)

	for i, comp := range r.Comps {
		if comp == nil {
			continue
		}
		compData := data.Get(pb.TypeComp(i))
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

func (r *Role) Run() {
	r.Wait.Add(1)
	thread.GoSafe(func() {
		defer func() {
			r.Offline()
			r.Wait.Done() // 最后Done
		}()
		r.Online()
		for {
			<-r.Events.Sig()
			r.Events.Range(func(evt Event) {
				if evt.Func != nil {
					evt.Func(r)
				} else {
					evt.Ctx.U = r
					if err := router.R().Handle(evt.Ctx); err != nil {
						zap.L().Warn("handle err", zap.Error(err))
					}
				}
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
	r.lastSaveRedis = now
	r.LastHeartbeat = now
	r.HeartbeatTimeOut = 0

	// 有些数据datareset需要先处理在发给客户端，避免客户端有1s收到头一天的数据
	r.OnTick(now)

	gnet.SendToCenter(msgid.MsgIDS2S_S2SReqLoginOrLogout, &pb.S2SReqLoginOrLogout{
		RoleID: r.ID,
		GameID: uint32(flag.SvcIndex),
		SesID:  r.SesID,
		Login:  true,
	}, pb.ActorID_IDGlobal)
	for _, v := range r.Comps {
		if comp, ok := v.(ICompOnline); ok {
			comp.Online(r)
		}
	}

	gnet.SendToGate(msgid.MsgIDS2S_S2SResLogin, &pb.S2SResLogin{
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

	for _, v := range r.Comps {
		if comp, ok := v.(ICompOffline); ok {
			comp.Offline(r)
		}
	}

	Mgr.Unregister(r.ID, r.SesID)

	data, err := r.marshal(true)
	if err == nil {
		LoginMgr().Logout(data) // offline时在mgr里保存,批量存
	}

	// 通知其它服务器
	r.Disconnect(pb.DisconnectReason_Kick)

	gnet.SendToCenter(msgid.MsgIDS2S_S2SReqLoginOrLogout, &pb.S2SReqLoginOrLogout{
		RoleID: r.ID,
		GameID: uint32(flag.SvcIndex),
		Login:  false,
	}, pb.ActorID_IDGlobal)
	zap.L().Info("[login] offline", zap.Inline(r))
}

func (r *Role) Disconnect(why pb.DisconnectReason) {
	gnet.SendToGate(msgid.MsgIDS2S_S2SS2GtDisconnect, &pb.S2SS2GtDisconnect{
		SesID: r.SesID,
		Why:   why,
	}, r.SesID)
}

// NowSec 当前时间，在组件中使用，避免每次使用time.now()。精确到秒，如需更高精度，不要使用
func (r *Role) NowSec() int64 {
	return r.nowSec
}

func (r *Role) OnTick(now time.Time) {
	if r.Data == nil {
		zap.L().Error("role.Data == nil")
		return
	}

	reset := false
	dayChange := false
	monthChange := false
	r.nowSec = now.Unix()

	if r.nowSec > r.Data.ResetTime {
		reset = true
		monthChange = r.nowSec >= r.Data.DataResetMonth
	}

	if r.nowSec > r.Data.DayChange {
		dayChange = true
	}

	if now.Sub(r.lastMinute) > time.Minute {
		r.lastMinute = now
		r.MinuteLoop(now)
	}

	for _, v := range r.Comps {
		if comp, ok := v.(ICompTick); ok {
			comp.OnTick(now, r)
		}

		if dayChange {
			if comp, ok := v.(ICompNewDay); ok {
				comp.OnNewDay(r)
			}
		}

		if reset { // 每日数据重置
			if comp, ok := v.(ICompDataReset); ok {
				comp.OnDataReset(r)
			}

			if monthChange {
				if comp, ok := v.(ICompNewMonth); ok {
					comp.OnNewMonth(r)
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
		}
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
	if now.Sub(r.lastSaveRedis).Seconds() > float64(conf.Time.AutoSave) {
		if now.Sub(r.lastSaveDB).Seconds() > float64(conf.Time.AutoSave*10) {
			r.save(true)
			r.lastSaveDB = now
		} else {
			r.save(false)
		}
		r.lastSaveRedis = now
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
	for _, v := range r.Comps {
		if iSec, ok := v.(ICompMinute); ok {
			iSec.OnMinute(now, r)
		}
	}
}

// GetComp	获取组件
func (r *Role) GetComp(t pb.TypeComp) IComp {
	return r.Comps[t]
}

// SendS	发送数据给客户端,存在反射,逃逸
func (r *Role) SendS(msg pb.VTMessage) {
	msgID, ok := pb.GetMsgIDS2C(msg)
	if !ok {
		zap.L().Error("[sendS] GetMsgIDS2C error")
		return
	}
	gnet.SendToClient(msgID, msg, r.SesID, r.ID)
}

// Send	发送数据给客户端,热路径建议使用Send
func Send[T pb.VTMessage](r *Role, msgID msgid.MsgIDS2C, msg T) {
	if util.Debug {
		realMsgID, ok := pb.GetMsgIDS2C(msg)
		if !ok {
			zap.L().Error("[SendT] GetMsgIDS2C error",
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
		if realMsgID != uint32(msgID) {
			zap.L().Error("[SendT] msgID mismatch",
				zap.Uint32("argMsgID", uint32(msgID)),
				zap.Uint32("realMsgID", realMsgID),
				zap.String("argMsgName", msgid.MsgIDS2C_name[int32(msgID)]),
				zap.String("type", fmt.Sprintf("%T", msg)),
			)
			return
		}
	}

	gnet.SendToClient(uint32(msgID), msg, r.SesID, r.ID)
}

func (r *Role) SetDirty() {
	r.dirty = true
}

func (r *Role) marshal(force bool) (*DataToSave, error) {
	rd := &DataToSave{
		RoleID: r.ID,
		Data:   make(map[string]string),
	}

	if force || r.dirty {
		str, err := sonic.MarshalString(r.Data)
		if err != nil {
			zap.S().Errorf("marshal role data err:%v", err)
			return nil, err
		}

		rd.Set(pb.TypeComp_TCBase, str)
	}

	for i, v := range r.Comps {
		if v == nil {
			continue
		}
		if !force && !v.IsDirty() {
			continue
		}
		str, err := sonic.MarshalString(v)
		if err != nil {
			zap.L().Error("marshal role comp data err", zap.Error(err), zap.Inline(r))
			continue
		}

		rd.Set(pb.TypeComp(i), str)
	}

	return rd, nil
}

func (r *Role) save(both bool) {
	data, err := r.marshal(false)
	if err != nil {
		return
	}
	if data.IsEmpty() {
		return
	}

	if LoginMgr().SaveRole(data, both) {
		r.dirty = false
		for _, v := range r.Comps {
			if v != nil && v.IsDirty() {
				v.ClearDirty()
			}
		}
	} else {
		zap.L().Warn("[save] role save failed")
	}
}
