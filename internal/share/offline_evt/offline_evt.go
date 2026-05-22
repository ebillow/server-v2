package offline_evt

import (
	"context"
	"fmt"
	"server/api/pb"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/thread"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type IRole interface {
	GetID() uint64
}

var offlineHandle = make(map[string]func(user IRole, msg proto.Message))

const (
	MaxEvents = 10
	Expire    = time.Hour * 24 * 30
)

func offlineDataKey(roleId uint64) string {
	return fmt.Sprintf("role:%d:%d:offline_evt", roleId, flag.SrvType)
}

// Add 玩家不在线时，添加离线事件
func Add(roleId uint64, msg proto.Message) {
	b, err := proto.Marshal(msg)
	if err != nil {
		zap.L().Error("AddOfflineData", zap.Error(err))
		return
	}

	evt := &pb.EvtOffline{
		MsgName: string(proto.MessageName(msg)),
		Data:    b,
		RoleID:  roleId,
	}
	dataB, err := proto.Marshal(evt)
	if err != nil {
		zap.S().Warnf("add offline event err:%v", err)
		return
	}
	key := offlineDataKey(roleId)

	pipe := db.Redis.Pipeline()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pipe.RPush(ctx, key, dataB)
	pipe.LTrim(ctx, key, -MaxEvents, -1)
	pipe.Expire(ctx, key, Expire)
	_, err = pipe.Exec(ctx)
	if err != nil {
		zap.L().Error("AddOfflineData", zap.Error(err), zap.Any("data", evt.String()))
		return
	}
	zap.S().Debugf("save offline event %d %s", evt.RoleID, evt.MsgName)
}

// Do 上线时处理离线事件
func Do(user IRole) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := offlineDataKey(user.GetID())
	ss, err := db.Redis.LRange(ctx, key, 0, MaxEvents-1).Result()
	if err != nil || len(ss) == 0 {
		return
	}

	for _, v := range ss {
		op := &pb.EvtOffline{}
		err = proto.Unmarshal([]byte(v), op)
		if err != nil {
			continue
		}
		t, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(op.MsgName))
		if err != nil {
			zap.S().Warnf("can not find offline proto type:%s", op.MsgName)
			continue
		}
		msg := t.New().Interface()
		if len(op.Data) > 0 {
			err = proto.Unmarshal(op.Data, msg)
			if err != nil {
				zap.S().Warnf("load offline op unmarshal err:%v", err)
				continue
			}
		}
		if f, ok := offlineHandle[op.MsgName]; !ok {
			zap.S().Warnf("can not find offline handler:%s", op.MsgName)
		} else {
			thread.RunSafe(func() {
				f(user, msg)
			})
			zap.S().Debugf("handle offline event:%d %s %v", op.RoleID, op.MsgName, msg)
		}
	}
	if len(ss) > 0 {
		db.Redis.LTrim(ctx, key, int64(len(ss)), -1)
	}
}

// RegisterHandler 注册离线事件处理方法
func RegisterHandler[T proto.Message](fn func(user IRole, msg T)) {
	if fn == nil {
		return
	}

	var msg T
	protoName := string(proto.MessageName(msg))

	offlineHandle[protoName] = func(u IRole, m proto.Message) {
		if mm, ok := m.(T); ok {
			fn(u, mm)
		}
	}
}
