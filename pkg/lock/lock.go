package lock

import (
	"context"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

var (
	rs *redsync.Redsync
)

// InitPool 使用锁之前，需调用一次初始化
func InitPool(redisCli redis.UniversalClient) {
	pool := goredis.NewPool(redisCli)
	rs = redsync.New(pool)
}

type Locker struct {
	mtx *redsync.Mutex
}

func NewLock(key string, opts ...redsync.Option) *Locker {
	defaultOps := []redsync.Option{
		redsync.WithExpiry(10 * time.Second),
		redsync.WithTries(3),
		redsync.WithRetryDelay(100 * time.Millisecond),
	}
	options := append(defaultOps, opts...)
	return &Locker{mtx: rs.NewMutex(key, options...)}
}

func (l *Locker) Lock(ctx context.Context) error {
	return l.mtx.LockContext(ctx)
}

func (l *Locker) Unlock(ctx context.Context) {
	_, err := l.mtx.UnlockContext(ctx)
	if err != nil {
		zap.L().Error("redsync unlock failed", zap.Error(err))
	}
}

// LockedDo 分布式锁，
func LockedDo(ctx context.Context, key string, fn func() error, opts ...redsync.Option) error {
	const lockKeyPrefix = "lock:"
	l := NewLock(lockKeyPrefix+key, opts...)
	err := l.Lock(ctx)
	if err != nil {
		return err
	}
	defer l.Unlock(ctx)

	return fn()
}
