package discovery

import (
	"github.com/stretchr/testify/require"
	"server/pkg/db"
	"server/pkg/logger"
	"server/pkg/pb"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	logger.NewZapLog("../../bin/log/test.log", logger.Config{
		Level:   -1,
		Console: true,
	})
	err := db.InitRedis(db.RedisCfg{
		Addr: []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
	}, 0)
	if err != nil {
		panic(err)
	}
	err = Init([]string{"127.0.0.1:2379"}, db.Redis)
	if err != nil {
		panic(err)
	}
	m.Run()
}

func TestNewWatcher(t *testing.T) {
	Watch()

	err := Register(pb.Server_Game, int32(1))
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 50)
	Close()
}

func TestRegister(t *testing.T) {
	err := Register(pb.Server_Game, int32(1))
	require.NoError(t, err)
	Watch()
	time.Sleep(time.Millisecond * 50)
	Close()
}
