package js

import (
	"context"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/pb"
	"strings"
	"time"
)

func (jt *JetStream) Serve(ctx context.Context, cb func(msg *pb.NatsMsg)) error {
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

func (jt *JetStream) sub(ctx context.Context, subject string, cb func(msg *pb.NatsMsg)) error {
	name := strings.ReplaceAll(subject, ".", "_")
	cfg := jetstream.StreamConfig{
		Name:        name,
		Description: name,
		Subjects:    []string{subject},
		// /		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:     24 * time.Hour, // 消息保留1天
		MaxBytes:   1024 * 1024 * 1024 * 10,
		Duplicates: time.Minute, // 1分钟内的消息去重窗口
	}

	// 幂等创建或更新 Stream
	stream, err := jt.JS.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return err
	}

	// 1. 创建或获取消费者 (Consumer)
	// 在 JetStream 中，Consumer 是服务器端的视图
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       name,                        // 持久化消费者名称
		AckPolicy:     jetstream.AckExplicitPolicy, // 显式确认
		FilterSubject: subject,                     // 监听所有日志
		MaxAckPending: 2000,                        // 流量控制：最多允许2000条未确认消息
		AckWait:       time.Minute,
	})
	if err != nil {
		return err
	}

	// 2. 开始消费消息 (Iterator / Consume 方法)
	// 使用 Consume 方法可以更方便地处理并发
	consContext, err := consumer.Consume(func(msgRaw jetstream.Msg) {
		// --- 业务逻辑开始 ---
		msg := &pb.NatsMsg{}
		err = proto.Unmarshal(msgRaw.Data(), msg)
		if err != nil {
			zap.L().Error("Error unmarshalling log", zap.Error(err))
			return
		}
		cb(msg)
		// --- 业务逻辑结束 ---

		// 3. 确认消息 (Ack)
		// 只有 Ack 后，Server 才会认为这条消息处理完成
		if err = msgRaw.Ack(); err != nil {
			zap.L().Error("Ack failed", zap.Error(err))
		}
	})
	if err != nil {
		return err
	}
	zap.L().Info("start consumer", zap.String("subject", subject))
	jt.consContext = append(jt.consContext, consContext)
	return nil
}
