package logic

import (
	session "server/gateway/session/v2"
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
			zap.L().Info("monitor", zap.Int32("链接人数", session.SessionCnt()))
		}
	}
}
