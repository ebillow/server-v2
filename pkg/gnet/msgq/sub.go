package msgq

import (
	"encoding/binary"
	"server/pkg/gerror"
	"server/pkg/gnet/codec"
	"server/pkg/gnet/gctx"
	"server/pkg/pb"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// BatchDecode 批量解码大包
func BatchDecode(buf []byte) ([]gctx.Context, error) {
	var ctxs []gctx.Context
	offset := 0

	for offset < len(buf) {
		// 1. 读取 4 字节的长度前缀
		if len(buf)-offset < 4 {
			return nil, gerror.New("batch decode error: missing length prefix")
		}
		subSize := int(binary.LittleEndian.Uint32(buf[offset:]))
		offset += 4

		// 2. 截取单条消息的数据段
		if len(buf)-offset < subSize {
			return nil, gerror.New("batch decode error: buffer too small for sub-message")
		}
		subBuf := buf[offset : offset+subSize]

		// 3. 复用你原来的 Decode 逻辑解析单条消息
		ctx, err := codec.Decode(subBuf)
		if err != nil {
			return nil, err // 或者记录错误并 continue
		}

		ctxs = append(ctxs, ctx)
		offset += subSize
	}

	return ctxs, nil
}

func (bs *DataBus) Serve(callback func(ctx gctx.Context)) error {
	err := bs.subscribe(bs.getSubjects(bs.serType, bs.serID), func(msg *nats.Msg) {
		// ctx, err := codec.Decode(msg.Data)
		// if err != nil {
		// 	zap.L().Warn("decode error", zap.Error(err))
		// 	return
		// }
		ctxs, err := BatchDecode(msg.Data)
		if err != nil {
			zap.L().Error("batch decode error", zap.Error(err))
		} else {
			for _, ctx := range ctxs {
				ctx.Raw = msg
				callback(ctx)
			}
		}
	})
	if err != nil {
		return err
	}

	return nil
}

func (bs *DataBus) Close() {
	err := bs.conn.Drain()
	if err != nil {
		zap.S().Warn("Failed to drain connection", zap.Error(err))
	}
	bs.conn.Close()
}

func (bs *DataBus) subscribe(subs map[string]string, callback func(msg *nats.Msg)) error {
	for sub, queue := range subs {
		if queue != "" {
			_, err := bs.conn.QueueSubscribe(sub, queue, callback)
			if err != nil {
				return err
			}
			zap.L().Info("queueSubscribe", zap.String("subject", sub), zap.String("queue", queue))
		} else {
			_, err := bs.conn.Subscribe(sub, callback)
			if err != nil {
				return err
			}
			zap.L().Info("subscribe", zap.Any("subject", sub))
		}
	}
	return nil
}

func (bs *DataBus) getSubjects(serType pb.Server, serID int32) map[string]string {
	subs := make(map[string]string)
	// all
	subs[getAllSubject(serType)] = ""
	// index
	subs[getIndexSubject(serType, serID)] = ""
	// group
	subs[getGroupSubject(serType)] = "msg.group"

	return subs
}
