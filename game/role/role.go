package role

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
	"server/pkg/flag"
	"server/pkg/gnet"
	"server/pkg/logger"
	"server/pkg/model"
	"server/pkg/thread"
	"server/pkg/util"
	"sync"

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

const EventChanSize = 128

type Event struct {
	Raw    *nats.Msg
	NatMsg *pb.NatsMsg

	CliMsg bool
	Func   func(r *Role)
}

// Role	角色数据
type Role struct {
	ID      uint64 // role_mgr需要访问
	SesID   uint64
	Comps   map[pb.TypeComp]IComp
	Data    *pb.RoleData // 入库数据
	CliInfo *pb.ClientInfo
	Seq     uint32

	Events chan Event
	Wait   sync.WaitGroup
	ctx    context.Context
	Cancel context.CancelFunc

	// 注意：临时属性，重连后就丢了
	FlagSave   bool
	NowSec     int64
	LastSave   time.Time
	LastMinute time.Time
}

var CreateComps func(r *Role)

// NewRole	新建一个角色
func NewRole(data *DataToSave, login *pb.S2SReqLogin) (*Role, error) {
	dataBase := &pb.RoleData{}

	err := jsoniter.UnmarshalFromString(data.Get(pb.TypeComp_TCBase), dataBase)
	if err != nil {
		return nil, err
	}

	r := &Role{
		ID:      data.ID,
		Data:    dataBase,
		SesID:   login.SesID,
		Comps:   make(map[pb.TypeComp]IComp),
		CliInfo: login.Req.CliInfo,
	}

	r.Events = make(chan Event, EventChanSize)
	r.ctx, r.Cancel = context.WithCancel(context.Background())

	CreateComps(r)

	for i, comp := range r.Comps {
		compData := data.Get(i)
		if len(compData) == 0 {
			continue
		}
		err = jsoniter.UnmarshalFromString(compData, comp)
		if err != nil {
			return nil, err
		}
	}

	if r.Data.CreateTime == 0 {

	}

	return r, nil
}

func (r *Role) CloseAndWait() {
	close(r.Events)
	r.Wait.Wait()
}

func (r *Role) Loop(ctx context.Context) {
	r.Wait.Add(1)
	thread.GoSafe(func() {
		t := time.NewTicker(time.Second)
		defer func() {
			r.Offline()
			t.Stop()
			r.Wait.Done() // 最后Done
		}()
		r.Online()
		for {
			select {
			case evt := <-r.Events:
				r.onEvent(evt)
			case now := <-t.C:
				r.SecLoop(now)
			case <-r.ctx.Done():
				return // 自己退出
			case <-ctx.Done():
				return // 进程退出
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
	r.Data.OnlineTime = time.Now().Unix()
	r.LastSave = time.Now()

	// 有些数据datareset需要先处理在发给客户端，避免客户端有1s收到头一天的数据
	// r.SecLoop(r)
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
	//
	// msgSend := &pb.S2CLogin{}
	// msgSend.Player = makeMsgForCli(r)
	// msgSend.GameID = setup.Setup.ID
	// msgSend.Dev = connParam.Dev
	// msgSend.Token = connParam.ReConnToken
	// msgSend.ServerNowTime = util.GetNowTimeM()
	// msgSend.ServerBeginTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local).Unix()
	//
	gnet.SendToGate(&pb.S2SResLogin{
		Res: &pb.S2CLogin{
			Player: r.Data,
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

	data, err := r.Marshal()
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

func (r *Role) Marshal() (*DataToSave, error) {
	rd := &DataToSave{
		ID:   r.ID,
		Data: make(map[string]string),
	}

	str, err := jsoniter.MarshalToString(r.Data)
	if err != nil {
		logger.Errorf("marshal role data err:%v", err)
		return nil, err
	}
	rd.Set(pb.TypeComp_TCBase, str)

	for i, v := range r.Comps {
		str, err = jsoniter.MarshalToString(v)
		if err != nil {
			zap.L().Error("marshal role comp data err", zap.Error(err), zap.Inline(r))
			continue
		}
		if len(str) == 0 {
			continue
		}

		rd.Set(i, str)
	}

	return rd, nil
}

func (r *Role) SecLoop(now time.Time) {
	if r.Data == nil {
		zap.L().Error("role.Data == nil")
		return
	}

	r.NowSec = time.Now().Unix()

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
			// logger.Debugf("%d data reset %v", r.Guid, time.Unix(r.Data.ResetTime, 0))
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
			logger.Debugf("%d data next month reset time=%v", r.ID, time.Unix(r.Data.DataResetMonth, 0))
		}
		// logger.Debugf("%d data reset time=%v", r.Guid, time.Unix(r.Data.ResetTime, 0))
	}
	if dayChange {
		begin := util.CurDayBegin()
		r.Data.DayChange = begin.Add(time.Duration(24) * time.Hour).Unix()
		// logger.Debugf("%d day change time=%v", r.Guid, time.Unix(r.Data.DayChange, 0))
		// r.Send(pb.MsgIDS2C_S2CDayChange, nil) // 告知客户端这一天过去了
	}
	if reset {
		// r.Send(pb.MsgIDS2C_S2CDataReset, nil) // 告知客户端数据重置
	}

	// if r.FlagSave || now.Sub(r.LastSave).Seconds() > float64(cfgs.Server().Config.CacheRoleTime) {
	// 	save(r.Data)
	// 	r.FlagSave = false
	// 	r.LastSave = now
	// }
}

func (r *Role) MinuteLoop(now time.Time) {
	for i := range r.Comps {
		if iSec, ok := r.Comps[i].(ICompMinuteLoop); ok {
			iSec.MinuteLoop(now, r)
		}
	}
}

func (r *Role) onEvent(evt Event) {
	if evt.NatMsg == nil {
		evt.Func(r)
	} else {
		r.onProto(evt.NatMsg, evt.Raw, evt.CliMsg)
	}
}

func (r *Role) onProto(natsMsg *pb.NatsMsg, raw *nats.Msg, isCli bool) {
	if isCli {
		cRouter().HandleWithRole(natsMsg, raw, r)
	} else {
		sRouter().HandleWithRole(natsMsg, raw, r)
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

func (r *Role) Save() {
	r.FlagSave = true
}
