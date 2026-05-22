package test

import (
	"server/api/pb/msgid"
	"server/pkg/logger"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	logger.NewZapLog("../../../../bin/log/test.log", logger.Config{
		Level:   0,
		Console: true,
	})
	m.Run()
}

func TestMsgID(t *testing.T) {
	zap.L().Info("test", zap.Any("msgid", msgid.MsgIDS2S_S2SGt2SDisconnect))
}

func TestZapDuration(t *testing.T) {
	now := time.Now()
	time.Sleep(time.Second)
	zap.L().Info("test", zap.Duration("duration", time.Since(now)))
}
