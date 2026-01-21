package util

import (
	"context"
	"go.uber.org/zap"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var luaIsMaster = redis.NewScript(`
if redis.call("set", KEYS[1], ARGV[1], "NX", "EX", ARGV[2]) then
	return redis.status_reply("OK")
end
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("set", KEYS[1], ARGV[1], "EX", ARGV[2])
end
`)

var luaDelMaster = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
`)

var isMaster atomic.Bool

func key(serName string) string {
	return "master:" + serName
}

// CheckAndSetMaster 检查并设置为主进程，抢占失败则返回false
func CheckAndSetMaster(cli redis.UniversalClient, ser string, val int32, t time.Duration) bool {
	_, err := luaIsMaster.Run(context.Background(), cli, []string{key(ser)}, val, t.Seconds()).Result()
	if err == nil {
		isMaster.Store(true)
	} else {
		isMaster.Store(false)
	}
	zap.L().Info("isMaster", zap.Bool("isMaster", isMaster.Load()))
	return err == nil
}

func IsMaster() bool {
	return isMaster.Load()
}

func DeleteMasterFlag(cli redis.UniversalClient, ser string, val int32) bool {
	zap.L().Debug("DeleteMasterFlag", zap.String("key", key(ser)), zap.Int32("val", val))
	_, err := luaDelMaster.Run(context.Background(), cli, []string{key(ser)}, val).Result()
	isMaster.Store(false)
	return err == nil
}
