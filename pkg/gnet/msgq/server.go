package msgq

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"server/pkg/flag"
	"server/pkg/pb"
)

func (bs *DataBus) Serve(callback func(wrapper *pb.NatsMsg, msg *nats.Msg)) error {
	err := bs.subscribe(bs.getSubjects(flag.SrvName(bs.serType), bs.serID), func(msg *nats.Msg) {
		wp, err := decode(msg.Data)
		if err != nil {
			zap.L().Warn("decode error", zap.Error(err))
			return
		}

		callback(wp, msg)
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

func (bs *DataBus) getSubjects(serName string, serID int32) map[string]string {
	subs := make(map[string]string)
	// all
	subs[getAllSubject(serName)] = ""
	// index
	subs[getIndexSubject(serName, serID)] = ""
	// group
	subs[getGroupSubject(serName)] = "msg.group"

	return subs
}
