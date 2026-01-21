package role

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/internal/log"
	"server/internal/util"
	"sync"

	"server/internal/pb"
	"time"
)

const EventChanSize = 128

type Event struct {
	MsgID uint32 // MsgID=0时，就是Func
	Data  []byte
	Func  func(r *Role)
}

// Role	角色数据
type Role struct {
	ID      uint64 // role_mgr需要访问
	SesID   uint64
	Comps   []IComp
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
func NewRole(dataStr string, login *pb.C2SLogin) (*Role, error) {
	data := &pb.RoleData{}
	err := jsoniter.UnmarshalFromString(dataStr, data)
	if err != nil {
		return nil, err
	}

	data.Channel = login.Channel

	r := &Role{
		ID:      data.ID,
		Data:    data,
		SesID:   login.SesID,
		Comps:   make([]IComp, pb.TypeComp_TCMax),
		CliInfo: login.CliInfo,
	}

	r.Events = make(chan Event, EventChanSize)
	r.ctx, r.Cancel = context.WithCancel(context.Background())

	CreateComps(r)

	if r.Data.CreateTime == 0 {
		onFirstInitData(r)
	}

	return r, nil
}

func (r *Role) CloseAndWait() {
	close(r.Events)
	r.Wait.Wait()
}

func (r *Role) Loop(ctx context.Context) {
	r.Wait.Add(1)
	util.GoSafe(func() {
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
				onEvent(evt, r)
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
	// for i := range r.Comps {
	// 	if iDeal, ok := r.Comps[i].(types.ICompOnline); ok {
	// 		iDeal.Online(r)
	// 	}
	// }
	//
	// msgSend := &pb.S2CLogin{}
	// msgSend.Player = makeMsgForCli(r)
	// msgSend.GameID = setup.Setup.ID
	// msgSend.Dev = connParam.Dev
	// msgSend.Token = connParam.ReConnToken
	// msgSend.ServerNowTime = util.GetNowTimeM()
	// msgSend.ServerBeginTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local).Unix()
	//
	// r.Send(msgSend)
	zap.L().Info("[login] online", zap.Uint64("id", r.ID), zap.String("name", r.Data.Name), zap.Uint32("cnt", r.Data.Exp))
}

// Offline 角色下线的处理
func (r *Role) Offline() {
	r.Data.OfflineTime = time.Now().Unix()
	for i := range r.Comps {
		if iDeal, ok := r.Comps[i].(ICompOffline); ok {
			iDeal.Offline(r)
		}
	}

	GetRoleMgr().Delete(r.ID, r.SesID)

	data, err := r.Marshal()
	if err != nil {
		return
	}
	GetLoginMgr().Offline(data) // offline时在mgr里保存,批量存

	// 通知其它服务器
	// network.SendToAllCenter(pb.MsgIDS2S_Gm2CtOffline, &pb.MsgKVGuidValue{Guid: r.Data.Guid, Value: setup.Setup.ID})
	zap.L().Info("[login] offline", zap.Uint64("id", r.ID), zap.String("name", r.Data.Name), zap.Uint32("cnt", r.Data.Country))
}

func (r *Role) Marshal() (*DataInDB, error) {
	rd := &DataInDB{
		ID: r.ID,
	}

	str, err := jsoniter.MarshalToString(r.Data)
	if err != nil {
		log.Errorf("marshal role data err:%v", err)
		return nil, err
	}
	rd.Data = str
	return rd, nil
}

func (r *Role) SecLoop(now time.Time) {
	//	zap.L().Debug("[role] sec loop")
	// if r.Data == nil {
	// 	log.Errorf("role.Data == nil")
	// 	return
	// }
	//
	// r.NowSec = time.Now().Unix()
	//
	// reset := false
	// dayChange := false
	// monthChange := false
	//
	// if r.NowSec > r.Data.ResetTime {
	// 	reset = true
	// 	monthChange = r.NowSec >= r.Data.DataResetMonth
	// }
	//
	// if r.NowSec > r.Data.DayChange {
	// 	dayChange = true
	// }
	//
	// if now.Sub(r.LastMinute) > time.Minute {
	// 	r.LastMinute = now
	// 	MinuteLoop(now, r)
	// }
	//
	// for i := range r.Comps {
	// 	if iSec, ok := r.Comps[i].(types.ICompSecLoop); ok {
	// 		iSec.SecLoop(now, r)
	// 	}
	//
	// 	if dayChange {
	// 		if iSec, ok := r.Comps[i].(types.ICompDayChange); ok {
	// 			iSec.OnDayChange(r)
	// 		}
	// 	}
	//
	// 	if reset { // 每日数据重置
	// 		// log.Debugf("%d data reset %v", r.Guid, time.Unix(r.Data.ResetTime, 0))
	// 		if iSec, ok := r.Comps[i].(types.ICompDataReset); ok {
	// 			iSec.OnDataReset(r)
	// 		}
	//
	// 		if monthChange {
	// 			if iSec, ok := r.Comps[i].(types.ICompMonthChange); ok {
	// 				iSec.OnMonthChange(r)
	// 			}
	// 		}
	// 	}
	// }
	//
	// if reset {
	// 	begin := util.CurDayBegin()
	// 	if now.Hour() >= int(cfgs.Server().GlobalSetup.ResetHour) {
	// 		r.Data.ResetTime = begin.Add(time.Duration(cfgs.Server().GlobalSetup.ResetHour+24) * time.Hour).Unix() // 下一次重置时间
	// 	} else {
	// 		r.Data.ResetTime = begin.Add(time.Duration(cfgs.Server().GlobalSetup.ResetHour) * time.Hour).Unix() // 下一次重置时间
	// 	}
	// 	if monthChange {
	// 		curMonthBegin := time.Date(now.Year(), now.Month(), 1, int(cfgs.Server().GlobalSetup.ResetHour), 0, 0, 0, now.Location())
	// 		r.Data.DataResetMonth = curMonthBegin.AddDate(0, 1, 0).Unix()
	// 		log.Debugf("%d data next month reset time=%v", r.Guid, time.Unix(r.Data.DataResetMonth, 0))
	// 	}
	// 	// log.Debugf("%d data reset time=%v", r.Guid, time.Unix(r.Data.ResetTime, 0))
	// }
	// if dayChange {
	// 	begin := util.CurDayBegin()
	// 	r.Data.DayChange = begin.Add(time.Duration(24) * time.Hour).Unix()
	// 	// log.Debugf("%d day change time=%v", r.Guid, time.Unix(r.Data.DayChange, 0))
	// 	r.Send(pb.MsgIDS2C_S2CDayChange, nil) // 告知客户端这一天过去了
	// }
	// if reset {
	// 	r.Send(pb.MsgIDS2C_S2CDataReset, nil) // 告知客户端数据重置
	// }
	//
	// if r.FlagSave || now.Sub(r.LastSave).Seconds() > float64(cfgs.Server().Config.CacheRoleTime) {
	// 	save(r.Data)
	// 	r.FlagSave = false
	// 	r.LastSave = now
	// }
}

func MinuteLoop(now time.Time, r *Role) {
	for i := range r.Comps {
		if iSec, ok := r.Comps[i].(ICompMinuteLoop); ok {
			iSec.MinuteLoop(now, r)
		}
	}
}

func onEvent(evt Event, r *Role) {
	if evt.MsgID == 0 {
		evt.Func(r)
	} else {
		onProto(evt.MsgID, evt.Data, r)
	}
}

func onProto(msgID uint32, data []byte, r *Role) {
	// zap.L().Debug("[role] onProto", zap.Uint32("id", msgID))
}

func onFirstInitData(r *Role) {
	for i := range r.Comps {
		if iDeal, ok := r.Comps[i].(ICompFirstInit); ok {
			iDeal.OnFirstInit(r)
		}
	}
}

// GetComp	获取组件
func (r *Role) GetComp(t pb.TypeComp) IComp {
	return r.Comps[t]
}

// Send	发送数据
func (r *Role) Send(msgData proto.Message) {

}

// SendBytes	发送Bytes数据
func (r *Role) SendBytes(msgID uint32, msgData []byte) {

}

func (r *Role) Save() {
	r.FlagSave = true
}
