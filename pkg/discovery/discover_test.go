package discovery

import (
	"github.com/stretchr/testify/require"
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
	err := Init([]string{"127.0.0.1:2379"})
	if err != nil {
		panic(err)
	}
	m.Run()
}

func TestNewRegistrar(t *testing.T) {
	err := Register(pb.Server_Game, 1)
	require.NoError(t, err)
	select {
	case <-time.After(time.Second):
		return
	}
}

func TestNewWatcher(t *testing.T) {
	Watch()

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second * 1)
		err := Register(pb.Server_Game, 1)
		require.NoError(t, err)
	}
}

func TestNewWatcherAfterRegistrar(t *testing.T) {
	go func() {
		for i := 0; i < 10; i++ {
			err := Register(pb.Server_Game, 1)
			require.NoError(t, err)
		}
	}()
	time.Sleep(time.Second * 1)
	Watch()
	select {
	case <-time.After(time.Second * 10):
	}
}

func TestService(t *testing.T) {
	Watch()

	for i := 0; i < 10; i++ {
		err := Register(pb.Server_Game, 1)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second * 1)
	}
	Close()
	time.Sleep(time.Second * 5)
}
