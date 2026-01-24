package msgq

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"server/internal/util"
	"strconv"
)

func (bs *DataBus) Serve(callback func(msg *nats.Msg)) error {
	err := bs.subscribe(bs.getSubjects(bs.serName, bs.serID), callback)
	if err != nil {
		return err
	}

	return nil
}

func (bs *DataBus) ServeCli(callback func(msg *nats.Msg)) error {
	subs := make(map[string]string)
	subs[getRoleSubject(bs.serName, util.ParseInt32(bs.serID))] = ""

	err := bs.subscribe(subs, callback)
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

func (bs *DataBus) getSubjects(serName string, serID string) map[string]string {
	subs := make(map[string]string)
	// all
	subs[getAllSubject(serName)] = ""
	// index
	idx, _ := strconv.Atoi(serID)
	subs[getIndexSubject(serName, int32(idx))] = ""
	// group
	subs[getGroupSubject(serName)] = "msg.group"

	return subs
}
