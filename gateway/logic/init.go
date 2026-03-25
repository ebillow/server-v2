package logic

import (
	"go.uber.org/zap"
	session "server/gateway/session/v2"
	"server/pkg/thread"
	"time"
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
