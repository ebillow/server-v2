package js

import (
	"context"
	"fmt"
	"server/pkg/flag"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"server/pkg/pb"
	"strings"
	"time"
)

func (jt *JetStream) Serve(ctx context.Context, cb func(msg *pb.NatsMsg)) error {
	if err := jt.initGlobalStream(ctx); err != nil {
		return err
	}
	subjects := []string{
		getIndexSubject(jt.serType, jt.serID),
		getGroupSubject(jt.serType)}

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
	streamName := getStreamName(jt.serType)
	wildcardSubject := getStreamWildcardSubject(jt.serType)

	cfg := jetstream.StreamConfig{
		Name:        streamName,
		Description: fmt.Sprintf("Unified stream for %s", flag.SrvName(jt.serType)),
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

func (jt *JetStream) sub(ctx context.Context, subject string, cb func(msg *pb.NatsMsg)) error {
	streamName := getStreamName(jt.serType)
	// 持久化消费者名称必须唯一，这里用 subject 转换 (例如: stream_game_idx_1)
	consumerName := strings.ReplaceAll(subject, ".", "_")

	// 1. 创建或获取消费者 (挂载到统一的 streamName 上)
	consumer, err := jt.JS.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:       consumerName,                // 持久化消费者名称
		AckPolicy:     jetstream.AckExplicitPolicy, // 显式确认
		FilterSubject: subject,                     // 【核心魔法】：只过滤出当前需要的 subject
		MaxAckPending: 2000,                        // 流量控制
		AckWait:       time.Minute,
	})
	if err != nil {
		zap.L().Error("Create consumer failed", zap.Error(err), zap.String("subject", subject))
		return err
	}

	// 2. 开始消费消息
	consContext, err := consumer.Consume(func(msgRaw jetstream.Msg) {
		msg := &pb.NatsMsg{}
		err = proto.Unmarshal(msgRaw.Data(), msg)
		if err != nil {
			zap.L().Error("Error unmarshalling log", zap.Error(err))
			msgRaw.Ack() // 解析失败直接丢弃，防止死循环
			return
		}

		// 执行业务回调
		cb(msg)

		// 处理完成，Ack 确认 todo ack放到cb里
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
