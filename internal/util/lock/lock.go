package lock

import (
	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	pool redsyncredis.Pool
)

// InitPool 使用锁之前，需调用一次初始化
func InitPool(redisCli redis.UniversalClient) {
	pool = goredis.NewPool(redisCli)
}

type Locker struct {
	mtx *redsync.Mutex
}

func NewLock(key string) *Locker {
	locker := redsync.New(pool)
	return &Locker{mtx: locker.NewMutex(key)}
}

func (l *Locker) Lock() error {
	return l.mtx.Lock()
}

func (l *Locker) Unlock() {
	_, err := l.mtx.Unlock()
	if err != nil {
		zap.L().Error("redsync unlock failed", zap.Error(err))
	}
}

// LockedDo 分布式锁，
func LockedDo(key string, fn func() error) error {
	l := NewLock("lock_" + key)
	err := l.Lock()
	if err != nil {
		return err
	}
	defer l.Unlock()

	return fn()
}
