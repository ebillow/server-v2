package lock

import (
	"server/pkg/db"
	"testing"
)

func TestMain(m *testing.M) {
	err := db.InitRedis(db.RedisCfg{
		Addr: []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
	}, 0)
	if err != nil {
		panic(err)
	}

	InitPool(db.Redis)
	m.Run()
}
