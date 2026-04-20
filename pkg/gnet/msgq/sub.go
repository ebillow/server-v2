package msgq

import (
	"server/pkg/gnet/codec"
	"server/pkg/gnet/gctx"
	"server/pkg/pb"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func (bs *DataBus) Serve(callback func(ctx gctx.Context)) error {
	err := bs.subscribe(bs.getSubjects(bs.serType, bs.serID), func(msg *nats.Msg) {
		ctx, err := codec.Decode(msg.Data)
		if err != nil {
			zap.L().Warn("decode error", zap.Error(err))
			return
		}
		ctx.Raw = msg
		callback(ctx)
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
