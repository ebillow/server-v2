package js

import (
	"context"
	"fmt"
	"server/api/pb"
	"server/pkg/flag"
	"server/pkg/gnet/pub"
	"strconv"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type JetStream struct {
	JS          jetstream.JetStream
	consContext []jetstream.ConsumeContext

	serType uint8
	serID   uint8

	closed atomic.Bool

	pub pub.Batchers[PubBatcher]

	ctx    context.Context
	cancel context.CancelFunc
}

var S JetStream

// Init 连接并确保 Stream 存在
func (jt *JetStream) Init(serType pb.Server, serID uint8, natsURL string, options ...nats.Option) error {
	jt.serType = uint8(serType)
	jt.serID = serID

	jt.ctx, jt.cancel = context.WithCancel(context.Background())

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

	// 创建 JetStream 上下文
	jt.JS, err = jetstream.New(nc)
	if err != nil {
		return err
	}

	jt.pub.Init(jt)

	return nil
}

func (jt *JetStream) Shutdown() {
	if !jt.closed.CompareAndSwap(false, true) {
		return
	}

	jt.pub.FlushAll()

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
	jt.cancel()
}

// 获取统一的 Stream 名称 (例如: Stream_game)
func getStreamName(serType pb.Server) string {
	return fmt.Sprintf("Stream_%s", flag.SrvName(serType))
}

// 获取 Stream 监听的通配符 Subject (例如: stream.game.>)
// ">" 在 NATS 中表示匹配多级后缀
func getStreamWildcardSubject(serType pb.Server) string {
	return fmt.Sprintf("stream.%s.>", flag.SrvName(serType))
}

// 获取具体节点的 Subject (例如: stream.game.idx.1)
func getIndexSubject(serType pb.Server, serID uint8) string {
	return fmt.Sprintf("stream.%s.idx.%d", flag.SrvName(serType), serID)
}

// 获取分组 Subject (例如: stream.game.group)
func getGroupSubject(serType pb.Server) string {
	return fmt.Sprintf("stream.%s.group", flag.SrvName(serType))
}
