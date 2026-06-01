package js

import (
	"context"
	"encoding/binary"
	"fmt"
	"server/api/pb"
	"server/pkg/flag"
	"server/pkg/gerror"
	"server/pkg/gnet/batcher"
	"server/pkg/gnet/gctx"

	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

func (jt *JetStream) Serve(ctx context.Context, cb func(msg gctx.Context)) error {
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

func (jt *JetStream) sub(ctx context.Context, subject string, cb func(msg gctx.Context)) error {
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
		err = BatchDecodeAndHandle(msgRaw.Data(), cb)

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

// BatchDecodeAndHandle 批量解码大包
func BatchDecodeAndHandle(buf []byte, cb func(msg gctx.Context)) error {
	offset := 0

	for offset < len(buf) {
		// 读取 4 字节的长度前缀
		if len(buf)-offset < 4 {
			return gerror.New("batch decode error: missing length prefix")
		}
		subSize := int(binary.LittleEndian.Uint32(buf[offset:]))
		offset += 4

		// 截取单条消息的数据段
		if len(buf)-offset < subSize {
			return gerror.New("batch decode error: buffer too small for sub-message")
		}
		subBuf := buf[offset : offset+subSize]

		ctx, err := Decode(subBuf)
		if err != nil {
			offset += subSize
			continue
		}

		cb(ctx)

		offset += subSize
	}

	return nil
}

func Decode(buf []byte) (ctx gctx.Context, err error) {
	if len(buf) < batcher.FrameBodyHeadSize {
		return ctx, gerror.New("decode error: buffer too small for header")
	}

	offset := 0

	ctx.MsgID = binary.LittleEndian.Uint32(buf[offset:])
	offset += 4

	ctx.ActorID = binary.LittleEndian.Uint64(buf[offset:])
	offset += 8

	ctx.SesID = binary.LittleEndian.Uint64(buf[offset:])
	offset += 8

	ctx.FromSer = buf[offset]
	offset += 1

	ctx.FromSerID = buf[offset]
	offset += 1

	ctx.ToSer = buf[offset]
	offset += 1

	ctx.ToSerID = buf[offset]
	offset += 1

	ctx.Flag = buf[offset]
	offset += 1

	ctx.Data = buf[offset:]

	return ctx, nil
}
