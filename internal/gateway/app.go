package gateway

import (
	"context"
	"server/internal/gateway/logic"
	"server/internal/gateway/session"
	"server/pkg/flag"
	"server/pkg/gnet/gmsg"
	"server/pkg/gnet/router"
	"server/pkg/util"
	"sync"
	"time"

	"go.uber.org/zap"
)

func Init(ctx context.Context) error {
	logic.Init()
	return nil
}

func Action(ctx context.Context, wait *sync.WaitGroup) error {
	cfg := loadNetCfg()
	session.StartWSServer("0.0.0.0:"+util.IToString(flag.TcpPort), cfg)
	return nil
}

func UnInit(ctx context.Context) {
	session.GracefulStop()
	zap.S().Info("server closed")
}

func loadNetCfg() *session.Config {
	d, err := time.ParseDuration("60s")
	if err != nil {
		zap.L().Error("parse read_dead_line err", zap.Error(err))
		return nil
	}
	cfg := &session.Config{
		ReadDeadline:        d,
		OutChanSize:         16,
		ReadSocketBuffSize:  1024,
		WriteSocketBuffSize: 1024,
		RecvPkgLenLimit:     uint32(10240),
	}
	zap.S().Infof("read_dead_line=%v, out_chan_size=%d, read_sock_size=%d, write_sock_size=%d,  pkg_len_limit=%d",
		cfg.ReadDeadline, cfg.OutChanSize, cfg.ReadSocketBuffSize, cfg.WriteSocketBuffSize, cfg.RecvPkgLenLimit)
	return cfg
}

func OnServerMsg(c gmsg.Message) {
	if c.Head.Flag == gmsg.Forward {
		ses := session.Get(c.Head.SesID)
		if ses == nil {
			return
		}
		ses.SendBytes(c.Head.MsgID, c.Data)
		return
	}

	err := router.R().Handle(c)
	if err != nil {
		zap.L().Info("<<< msg.recv:",
			zap.Inline(&c),
		)
	}
}
