package db

import (
	"context"
	"server/pkg/gerror"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

/*
集群模式，可以使用pipe，不能使用同时访问多个key的命令：
1.集合运算：SUNION, SINTER, SDIFF已经对应的STORE
2.RPOPLPUSH, BRPOPLPUSH, LMOVE, BLMOVE
3.ZUNION, ZDIFF, ZINTER以及对应的STORE， WITHSOCRE
4.MGET，MSET
5.RENAME,RENAMENX
6.LUA脚本中访问的key必须在同一个Slot
7.SCAN只能扫描当前节点的key
8.FLUSHDB，DBSIZE当前节点
*/

type RedisCfg struct {
	Addr     []string
	Password string
	DB       int // 集群模式，只能=0
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
	// 生产环境强制防御：集群模式 DB 必须为 0
	if len(conf.Addr) > 1 && conf.DB != 0 {
		zap.L().Warn("Redis cluster mode detected, resetting DB to 0", zap.Int("original_db", conf.DB))
		conf.DB = 0
	}

	cli := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:      conf.Addr,
		ClientName: conf.Name,
		DB:         conf.DB,
		Password:   conf.Password,

		PoolTimeout:  5 * time.Second,
		PoolSize:     poolSize,
		MinIdleConns: poolSize / 4,

		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cli.Ping(ctx).Err(); err != nil {
		return nil, gerror.Wrap(err, "redis ping failed")
	}

	return cli, nil
}

func CloseRedis() {
	if Redis != nil {
		_ = Redis.Close()
	}
}
