package js

import (
	"context"
	"fmt"
	"server/api/pb"
	"server/pkg/flag"
	"server/pkg/gnet/pkg"

	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

func (jt *JetStream) Serve(ctx context.Context, cb func(msg pkg.Packet)) error {
	if err := jt.initGlobalStream(ctx); err != nil {
		return err
	}
	subjects := []string{
		getIndexSubject(pb.Server(jt.serType), jt.serID),
		getGroupSubject(pb.Server(jt.serType))}

	for _, subject := range subjects {
		err := jt.sub(ctx, subject, cb)
		if err != nil {
			return err
		}
	}
	return nil
}

// initGlobalStream 幂等创建或更新全局统一的 Stream
func (jt *JetStream) initGlobalStream(ctx context.Context) error {
	streamName := getStreamName(pb.Server(jt.serType))
	wildcardSubject := getStreamWildcardSubject(pb.Server(jt.serType))

	cfg := jetstream.StreamConfig{
		Name:        streamName,
		Description: fmt.Sprintf("Unified stream for %s", flag.SrvName(pb.Server(jt.serType))),
		Subjects:    []string{wildcardSubject}, // 核心：监听通配符
		MaxAge:      24 * time.Hour,            // 消息保留1天
		MaxBytes:    10 * 1024 * 1024 * 1024,   // 10GB 限制
		Duplicates:  time.Minute,               // 1分钟内的消息去重窗口
	}

	// 幂等操作：如果不存在则创建，如果存在且配置不同则更新
	_, err := jt.JS.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		zap.L().Error("Failed to init global stream", zap.Error(err), zap.String("stream", streamName))
		return err
	}
	zap.L().Info("Global stream initialized successfully", zap.String("stream", streamName))
	return nil
}

func (jt *JetStream) sub(ctx context.Context, subject string, cb func(msg pkg.Packet)) error {
	streamName := getStreamName(pb.Server(jt.serType))
	// 持久化消费者名称必须唯一，这里用 subject 转换 (例如: stream_game_idx_1)
	consumerName := strings.ReplaceAll(subject, ".", "_")

	//  创建或获取消费者 (挂载到统一的 streamName 上)
	consumer, err := jt.JS.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:       consumerName,                // 持久化消费者名称
		AckPolicy:     jetstream.AckExplicitPolicy, // 显式确认
		FilterSubject: subject,                     // 只过滤出当前需要的 subject
		MaxAckPending: 10000,                       // 流量控制
		AckWait:       time.Minute,
	})
	if err != nil {
		zap.L().Error("Create consumer failed", zap.Error(err), zap.String("subject", subject))
		return err
	}

	// 开始消费消息
	consContext, err := consumer.Consume(func(msgRaw jetstream.Msg) {
		err = pkg.DecodeManyAndHandle(msgRaw.Data(), msgRaw.Subject(), msgRaw.Reply(), cb)

		// 处理完成，Ack 确认
		if err = msgRaw.Ack(); err != nil {
			zap.L().Error("Ack failed", zap.Error(err))
		}
	})
	if err != nil {
		return err
	}

	zap.L().Info("Start consumer", zap.String("stream", streamName), zap.String("subject", subject))
	jt.consContext = append(jt.consContext, consContext)
	return nil
}
