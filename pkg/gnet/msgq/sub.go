package msgq

import (
	"encoding/binary"
	"server/pkg/gerror"
	"server/pkg/gnet/gctx"
	"server/pkg/pb"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func (bs *DataBus) Serve(callback func(ctx gctx.Context)) error {
	err := bs.subscribe(bs.getSubjects(pb.Server(bs.serType), bs.serID), func(msg *nats.Msg) {
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

func (bs *DataBus) getSubjects(serType pb.Server, serID uint8) map[string]string {
	subs := make(map[string]string) // [subject,queue]
	// all
	subs[getAllSubject(serType)] = ""
	// index
	subs[getIndexSubject(serType, serID)] = ""
	// group
	subs[getGroupSubject(serType)] = "msg.group"
	// rpc idx
	subs[getRpcIdxSubject(serType, serID)] = ""

	return subs
}

// BatchDecode 批量解码大包
func BatchDecode(buf []byte) ([]gctx.Context, error) {
	var ctxs []gctx.Context
	offset := 0

	for offset < len(buf) {
		// 读取 4 字节的长度前缀
		if len(buf)-offset < 4 {
			return nil, gerror.New("batch decode error: missing length prefix")
		}
		subSize := int(binary.LittleEndian.Uint32(buf[offset:]))
		offset += 4

		// 截取单条消息的数据段
		if len(buf)-offset < subSize {
			return nil, gerror.New("batch decode error: buffer too small for sub-message")
		}
		subBuf := buf[offset : offset+subSize]

		ctx, err := Decode(subBuf)
		if err != nil {
			return nil, err // 或者记录错误并 continue
		}

		ctxs = append(ctxs, ctx)
		offset += subSize
	}

	return ctxs, nil
}

func Decode(buf []byte) (ctx gctx.Context, err error) {
	if len(buf) < headerSize {
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
