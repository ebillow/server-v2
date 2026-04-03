package logger

import (
	"go.uber.org/zap"
	"testing"
)

func TestLog(t *testing.T) {
	NewZapLog("../../bin/logger/test.logger", Config{
		Level:     0,
		Console:   true,
		NoticeUrl: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=1fb68582-e121-4a95-8c26-88dc9fd27c91",
	})

	zap.L().Info("info msg", zap.Int("int", 123))
	zap.L().Warn("warn msg", zap.Int("int", 123))
	zap.L().Error("test error msg", zap.Int("int", 123))
}
