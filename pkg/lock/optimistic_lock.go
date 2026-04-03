package lock

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"math/rand"
	"time"
)

// OptimisticLockedDo 乐观锁，保存时可以用save这个pipe
func OptimisticLockedDo(ctx context.Context, redisCli redis.UniversalClient, key string, fn func(ctx context.Context, tx *redis.Tx, save redis.Pipeliner) error) error {
	txfTarget := func(tx *redis.Tx) error {
		pipe := tx.TxPipeline()
		err := fn(ctx, tx, pipe)
		if err != nil {
			return err
		}
		_, err = pipe.Exec(ctx)
		return err
	}

	const maxRetries = 5

	for i := 0; i < maxRetries; i++ {
		err := redisCli.Watch(ctx, txfTarget, key)
		if err == nil {
			break
		}
		if errors.Is(err, redis.TxFailedErr) {
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)+5))
			zap.L().Debug("lock tx failed", zap.String("key", key), zap.Error(err))
			continue
		}
		return err
	}

	return nil
}
