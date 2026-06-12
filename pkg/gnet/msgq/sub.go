package msgq

import (
	"server/api/pb"
	"server/pkg/gnet/gmsg"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func (bs *DataBus) Serve(callback func(msg gmsg.Message)) error {
	err := bs.subscribe(bs.getSubjects(pb.Server(bs.serType), bs.serID), func(msg *nats.Msg) {
		err := gmsg.DecodeManyAndHandle(msg.Data, msg.Subject, msg.Reply, callback)
		if err != nil {
			zap.L().Error("batch decode error", zap.Error(err))
		}
	})
	if err != nil {
		return err
	}

	return nil
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

func (bs *DataBus) Close() {
	if !bs.closed.CompareAndSwap(false, true) {
		return
	}

	bs.pub.FlushAll()

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

func (bs *DataBus) getSubjects(serType pb.Server, serID uint8) map[string]string {
	subs := make(map[string]string) // [subject,queue]
	// all
	subs[broadcastSubjectName(serType)] = ""
	// index
	subs[nodeSubjectName(serType, serID)] = ""
	// group
	subs[anySubjectName(serType)] = "msg.group"
	// rpc idx
	subs[rpcNodeSubjectName(serType, serID)] = ""

	return subs
}
