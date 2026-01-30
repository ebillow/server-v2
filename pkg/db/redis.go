package db

import (
	"context"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

type RedisCfg struct {
	Addr     []string
	Password string
	DB       int
	Name     string
}

var Redis redis.UniversalClient // 通用业务 redis 客户端

// InitRedis 初始化业务 redis 客户端
func InitRedis(conf RedisCfg, poolSize int) error {
	cli, err := NewRedis(conf, poolSize)
	if err != nil {
		return err
	}
	Redis = cli
	zap.L().Info("redis connected", zap.Any("addr", conf.Addr), zap.Int("acc_db", conf.DB))
	return nil
}

// NewRedis 创建一个 redis 客户端
func NewRedis(conf RedisCfg, poolSize int) (redis.UniversalClient, error) {
	const DefaultPoolSizeRedis = 10
	if poolSize <= 0 {
		poolSize = DefaultPoolSizeRedis
	}

	cli := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:           conf.Addr,
		ClientName:      conf.Name,
		DB:              conf.DB,
		Password:        conf.Password,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        poolSize,
	})

	if err := cli.Ping(context.Background()).Err(); err != nil {
		return nil, errors.Wrap(err, "redis ping failed")
	}

	return cli, nil
}
