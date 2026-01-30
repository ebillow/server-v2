package lock

import (
	"context"
	"github.com/redis/go-redis/v9"
	"server/pkg/idgen"
	"time"
)

type SimpleLock struct {
	client     redis.UniversalClient
	lockKey    string
	lockValue  int64
	expiration time.Duration
}

// NewSimpleLock 简单分布式锁，锁失败不等待，立即返回false
func NewSimpleLock(client redis.UniversalClient, lockKey string, expiration time.Duration) *SimpleLock {
	return &SimpleLock{
		client:     client,
		lockKey:    lockKey,
		lockValue:  idgen.Gen(),
		expiration: expiration,
	}
}

func (l *SimpleLock) Lock(ctx context.Context) (bool, error) {
	result, err := l.client.SetNX(ctx, l.lockKey, l.lockValue, l.expiration).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

func (l *SimpleLock) Unlock(ctx context.Context) error {
	script := `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end`

	_, err := l.client.Eval(ctx, script, []string{l.lockKey}, l.lockValue).Result()
	return err
}
