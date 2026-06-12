package offline_evt

import (
	"context"
	"fmt"
	"server/api/pb"
	"server/internal/game/role"
	"server/pkg/db"
	"server/pkg/flag"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/router"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func offlineMsgKey(roleId uint64) string {
	return fmt.Sprintf("role:%d:%d:offline_msg", roleId, flag.SrvType)
}

// AddMsg 玩家不在线时，添加离线消息
func AddMsg(msg *pb.NatsMsg) {
	dataB, err := proto.Marshal(msg)
	if err != nil {
		zap.S().Warnf("add offline event err:%v", err)
		return
	}
	key := offlineMsgKey(msg.RoleID)

	pipe := db.Redis.Pipeline()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pipe.RPush(ctx, key, dataB)
	pipe.LTrim(ctx, key, -MaxEvents, -1)
	pipe.Expire(ctx, key, Expire)
	_, err = pipe.Exec(ctx)
	if err != nil {
		zap.L().Error("AddOfflineData", zap.Error(err), zap.Any("data", msg.String()))
		return
	}
	zap.S().Debugf("save offline event %d %s", msg.RoleID, msg.String())
}

func HandleMsg(r *role.Role) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := offlineDataKey(r.ID)
	ss, err := db.Redis.LRange(ctx, key, 0, MaxEvents-1).Result()
	if err != nil || len(ss) == 0 {
		return
	}

	for _, v := range ss {
		msg := &pb.NatsMsg{}
		err = proto.Unmarshal([]byte(v), msg)
		if err != nil {
			continue
		}

		router.S().Handle(gmsg.Message{
			Data:    msg.Data,
			U:       r,
			ActorID: r.ID,
			MsgID:   msg.MsgID,
		})
	}
	if len(ss) > 0 {
		db.Redis.LTrim(ctx, key, int64(len(ss)), -1)
	}
}
