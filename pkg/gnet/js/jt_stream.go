package js

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"server/pkg/flag"
	"server/pkg/pb"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type JetStream struct {
	JS          jetstream.JetStream
	consContext []jetstream.ConsumeContext
	ack         chan jetstream.PubAckFuture

	serType pb.Server
	serID   int32
}

var S JetStream

// Init 连接并确保 Stream 存在
func (jt *JetStream) Init(serType pb.Server, serID int32, natsURL string, options ...nats.Option) error {
	jt.serType = serType
	jt.serID = serID

	// 1. 配置 NATS 连接选项 (生产环境必须配置重连逻辑)
	opts := append(
		options,
		nats.Name(flag.SrvName(serType)+strconv.Itoa(int(jt.serID))),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(100),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			zap.S().Errorf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			zap.L().Info("NATS reconnected")
		}),
	)

	nc, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return err
	}

	// 2. 创建 JetStream 上下文
	jt.JS, err = jetstream.New(nc)
	if err != nil {
		return err
	}

	return nil
}

func (jt *JetStream) Shutdown() {
	if jt.JS != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		select {
		case <-jt.JS.PublishAsyncComplete():
			zap.L().Info("All async messages published")
		case <-ctx.Done():
			zap.L().Warn("Timeout waiting for async publish complete")
		}
	}
	for _, c := range jt.consContext {
		c.Stop()
	}
}

func getIndexSubject(serType pb.Server, serID int32) string {
	return fmt.Sprintf("stream.%s.idx.%d", flag.SrvName(serType), serID)
}

func getGroupSubject(serType pb.Server) string {
	return fmt.Sprintf("stream.%s.group", flag.SrvName(serType))
}
