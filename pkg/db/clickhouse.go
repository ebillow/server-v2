package db

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var ck driver.Conn

func CK() driver.Conn {
	return ck
}

func InitClickhouse(addr []string, usr string, pwd string, dbname string) error {
	var err error
	ck, err = clickhouse.Open(&clickhouse.Options{
		Addr: addr, // 非加密端口 9000
		Auth: clickhouse.Auth{
			Database: dbname,
			Username: usr,
			Password: pwd,
		},
		DialTimeout: 10 * time.Second,
		ReadTimeout: 10 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		Settings: clickhouse.Settings{
			"max_execution_time": 60, // 限制单次查询最大执行时间
		},
	})
	if err != nil {
		return err
	}

	// 测试连接
	if err := ck.Ping(context.Background()); err != nil {
		return err
	}
	return nil
}

func CloseDb() {
	if ck != nil {
		_ = ck.Close()
	}
}
