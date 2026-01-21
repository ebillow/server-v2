package log

import (
	"go.uber.org/zap"
	"testing"
)

func TestMain(m *testing.M) {
	Init("/Users/tao.zhou/Documents/soft/filebeat-elastic/logs/test.log", &Config{
		Level:      "debug",
		Console:    true,
		MaxSize:    100,
		MaxBackups: 30,
		MaxAge:     7,
		Compress:   false,
	})
	m.Run()
}

func TestLog(t *testing.T) {
	zap.L().With(zap.String("name", "name")).Info("test log")
}
