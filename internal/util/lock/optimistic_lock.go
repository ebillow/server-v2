package lock

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

const maxRetries = 100

// Do 乐观锁，保存时，需要用tx.TxPipeline(),并且返回error
func Do(redisCli redis.UniversalClient, key string, fn func(ctx context.Context, tx *redis.Tx) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()
	txfTarget := func(tx *redis.Tx) error {
		return fn(ctx, tx)
	}

	//快要上线，全部改逻辑风险大，先这样处理
	lock := NewLock("lock_" + key)
	err := lock.Lock()
	if err != nil {
		return err
	}
	defer lock.Unlock()

	for i := 0; i < maxRetries; i++ {
		err := redisCli.Watch(ctx, txfTarget, key)
		if err == nil {
			break
		}
		if errors.Is(err, redis.TxFailedErr) {
			time.Sleep(time.Millisecond)
			zap.L().Debug("lock tx failed", zap.String("key", key), zap.Error(err))
			continue
		}
		return err
	}
	return nil
}

// DoWithSavePipe 乐观锁，保存时可以用save这个pipe
func DoWithSavePipe(redisCli redis.UniversalClient, key string, fn func(ctx context.Context, tx *redis.Tx, save redis.Pipeliner) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()
	txfTarget := func(tx *redis.Tx) error {
		pipe := tx.TxPipeline()
		err := fn(ctx, tx, pipe)
		if err != nil {
			return err
		}
		_, err = pipe.Exec(ctx)
		return err
	}

	//快要上线，全部改逻辑风险大，先这样处理
	lock := NewLock("lock_" + key)
	err := lock.Lock()
	if err != nil {
		return err
	}
	defer lock.Unlock()

	for i := 0; i < maxRetries; i++ {
		err := redisCli.Watch(ctx, txfTarget, key)
		if err == nil {
			break
		}
		if errors.Is(err, redis.TxFailedErr) {
			time.Sleep(time.Millisecond * 10)
			zap.L().Debug("lock tx failed", zap.String("key", key), zap.Error(err))
			continue
		}
		return err
	}
	return nil
}
