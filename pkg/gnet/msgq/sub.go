package msgq

import (
	"encoding/binary"
	"server/api/pb"
	"server/pkg/gerror"
	"server/pkg/gnet/gctx"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func (bs *DataBus) Serve(callback func(ctx gctx.Context)) error {
	err := bs.subscribe(bs.getSubjects(pb.Server(bs.serType), bs.serID), func(msg *nats.Msg) {
		err := BatchDecodeAndHandle(msg, callback)
		if err != nil {
			zap.L().Error("batch decode error", zap.Error(err))
		}
	})
	if err != nil {
		return err
	}

	return nil
}

func (bs *DataBus) Close() {
	if !bs.closed.CompareAndSwap(false, true) {
		return
	}

	bs.flushAllBatchers()

	if bs.conn != nil {
		err := bs.conn.Drain()
		if err != nil {
			zap.S().Warn("Failed to drain connection", zap.Error(err))
		}
		bs.conn.Close()
	}
	if bs.rpcConn != nil {
		err := bs.rpcConn.Drain()
		if err != nil {
			zap.S().Warn("Failed to drain connection", zap.Error(err))
		}
		bs.rpcConn.Close()
	}
}

func (bs *DataBus) subscribe(subs map[string]string, callback func(msg *nats.Msg)) error {
	for sub, queue := range subs {
		var subObj *nats.Subscription
		var err error
		if queue != "" {
			subObj, err = bs.conn.QueueSubscribe(sub, queue, callback)
			if err != nil {
				return err
			}
			zap.L().Info("queueSubscribe", zap.String("subject", sub), zap.String("queue", queue))
		} else {
			subObj, err = bs.conn.Subscribe(sub, callback)
			if err != nil {
				return err
			}
			zap.L().Info("subscribe", zap.Any("subject", sub))
		}
		// 消息数量限制 (DefaultSubPendingMsgsLimit)：65,536 条 (64K)
		// 消息字节限制 (DefaultSubPendingBytesLimit)：67,108,864 字节 (64MB)
		err = subObj.SetPendingLimits(500000, 256*1024*1024)
		if err != nil {
			zap.L().Error("SetPendingLimits", zap.Error(err))
		}
	}
	return nil
}

func (bs *DataBus) getSubjects(serType pb.Server, serID uint8) map[string]string {
	subs := make(map[string]string) // [subject,queue]
	// all
	subs[allSubjectName(serType)] = ""
	// index
	subs[idxSubjectName(serType, serID)] = ""
	// group
	subs[groupSubjectName(serType)] = "msg.group"
	// rpc idx
	subs[rpcIdxSubjectName(serType, serID)] = ""

	return subs
}

// BatchDecodeAndHandle 批量解码大包
func BatchDecodeAndHandle(msg *nats.Msg, callback func(ctx gctx.Context)) error {
	buf := msg.Data
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
			continue // 或者记录错误并 continue
		}

		ctx.Raw = msg
		callback(ctx)
		offset += subSize
	}

	return nil
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
