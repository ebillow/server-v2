package test

import (
	"go.uber.org/zap"
	"server/pkg/logger"
	"server/pkg/pb/msgid"
	"testing"
	"time"
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
