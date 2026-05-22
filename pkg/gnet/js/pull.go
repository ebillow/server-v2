package js

import (
	"context"
	"server/api/pb"
	"strings"
	"time"

	"server/pkg/gnet/gctx"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type PullConsumer struct {
	JS       jetstream.JetStream
	Consumer jetstream.Consumer
}

// NewPullConsumer 初始化 Pull 消费者
func NewPullConsumer(ctx context.Context, jt *JetStream, subject string) (*PullConsumer, error) {
	if err := jt.initGlobalStream(ctx); err != nil {
		return nil, err
	}
	streamName := getStreamName(pb.Server(jt.serType))
	// 持久化消费者名称必须唯一，这里用 subject 转换 (例如: stream_game_idx_1)
	consumerName := strings.ReplaceAll(subject, ".", "_")

	// 创建或获取消费者 (挂载到统一的 streamName 上)
	consumer, err := jt.JS.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:       consumerName,                // 持久化消费者名称
		AckPolicy:     jetstream.AckExplicitPolicy, // 显式确认
		FilterSubject: subject,                     // 只过滤出当前需要的 subject
		MaxAckPending: 2000,                        // 流量控制
		AckWait:       time.Minute,
	})
	if err != nil {
		zap.L().Error("Create consumer failed", zap.Error(err), zap.String("subject", subject))
		return nil, err
	}

	return &PullConsumer{JS: jt.JS, Consumer: consumer}, nil
}

// Start 开始阻塞拉取并写入
func (c *PullConsumer) Start(ctx context.Context, cb func([]gctx.Context) error) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				zap.L().Info("DBLog consumer stopped")
				return
			default:
				// 每次拉取最多 50 个大包（假设每个大包 500 条，就是 25000 条），最多等 2 秒
				batchSize := 50
				timeout := 2 * time.Second

				// Fetch 会阻塞等待
				msgBatch, err := c.Consumer.Fetch(batchSize, jetstream.FetchMaxWait(timeout))
				if err != nil {
					time.Sleep(100 * time.Millisecond) // 超时或无消息，稍微退避
					continue
				}

				var allLogs []gctx.Context
				var natsMsgs []jetstream.Msg

				// 遍历拉取到的大包并解包
				for msg := range msgBatch.Messages() {
					ctxs, err := BatchDecode(msg.Data())
					if err != nil {
						zap.L().Error("DBLog batch decode error", zap.Error(err))
						msg.Ack() // 脏数据直接 Ack 丢弃
						continue
					}

					allLogs = append(allLogs, ctxs...)
					natsMsgs = append(natsMsgs, msg)
				}

				if len(allLogs) == 0 {
					continue
				}

				err = cb(allLogs)
				if err != nil {
					zap.L().Error("Failed to write to ClickHouse", zap.Error(err))
					// 写入失败不 Ack，等待 AckWait 超时后 NATS 会自动重发
					time.Sleep(2 * time.Second)
					continue
				}

				// 写入成功，批量 Ack NATS 的大包
				for _, msg := range natsMsgs {
					msg.Ack()
				}
				zap.L().Info("Successfully wrote logs to ClickHouse", zap.Int("log_count", len(allLogs)))
			}
		}
	}()
}
