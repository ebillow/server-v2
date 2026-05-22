package logic

import (
	"server/internal/gateway/session"
	"server/pkg/thread"
	"time"

	"go.uber.org/zap"
)

func Init() {
	thread.GoSafe(func() {
		Monitor()
	})
}

func Monitor() {
	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-t.C:
			zap.L().Info("monitor", zap.Int("链接人数", session.Count()))
		}
	}
}
