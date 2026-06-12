package msgq

import (
	"server/api/pb"
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gmetrics"
	"server/pkg/gnet/pkg"
	"server/pkg/gnet/pub"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// SendTo 发给指定服务
func (bs *DataBus) SendTo(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}
	pbt, err := bs.pub.GetIdx(serType, serID)
	if err != nil {
		return err
	}
	return pbt.Add(pkg.Packet{
		Head: pkg.Head{
			MsgID:     msgID,
			FromSerID: bs.serID,
			FromSer:   bs.serType,
			ToSer:     uint8(serType),
			ToSerID:   serID,
			ActorID:   actorID,
			SesID:     sesID,
		},
		Data: data,
	})
}

func (bs *DataBus) ForwardToClient(serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}

	pbt, err := bs.pub.GetIdx(pb.Server_Gateway, serID)
	if err != nil {
		return err
	}
	return pbt.Add(pkg.Packet{
		Head: pkg.Head{
			MsgID:     msgID,
			FromSerID: bs.serID,
			FromSer:   bs.serType,
			ToSer:     uint8(pb.Server_Gateway),
			ToSerID:   serID,
			ActorID:   actorID,
			SesID:     sesID,
			Flag:      pkg.Forward,
		},
		Data: data,
	})
}

// SendToAny 组发送. 随机一个能收到
func (bs *DataBus) SendToAny(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}

	pbt, err := bs.pub.GetGroup(serType)
	if err != nil {
		return err
	}

	return pbt.Add(pkg.Packet{
		Head: pkg.Head{
			MsgID:     msgID,
			FromSer:   bs.serType,
			FromSerID: bs.serID,
			ToSer:     uint8(serType),
			ActorID:   actorID,
			SesID:     sesID,
		},
		Data: data,
	})
}

// SendToGroup 同类型所有的服节点都能收到
func (bs *DataBus) SendToGroup(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}

	pbt, err := bs.pub.GetAll(serType)
	if err != nil {
		return err
	}

	return pbt.Add(pkg.Packet{
		Head: pkg.Head{
			MsgID:     msgID,
			FromSer:   bs.serType,
			FromSerID: bs.serID,
			ToSer:     uint8(serType),
			ActorID:   actorID,
			SesID:     sesID,
		},
		Data: data,
	})
}

// RelayTo 转发给指定服
func (bs *DataBus) RelayTo(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}

	pbt, err := bs.pub.GetGroup(pb.Server_Center)
	if err != nil {
		return err
	}

	return pbt.Add(pkg.Packet{
		Head: pkg.Head{
			MsgID:     msgID,
			FromSer:   bs.serType,
			FromSerID: bs.serID,
			ToSer:     uint8(serType),
			ToSerID:   serID,
			ActorID:   actorID,
			SesID:     sesID,
			Flag:      pkg.Forward,
		},
		Data: data,
	})
}

type BatcherConfig struct {
	MaxCount      int           // 最大条数
	MaxBytes      int           // 最大字节数
	FlushInterval time.Duration // 最大延迟
}

// 不同场景不同配置
var (
	RealtimeConfig   = BatcherConfig{MaxCount: 50, MaxBytes: 64 * 1024, FlushInterval: 5 * time.Millisecond}
	ThroughputConfig = BatcherConfig{MaxCount: 500, MaxBytes: 512 * 1024, FlushInterval: 25 * time.Millisecond}
)

// PubBatcher 针对单个目标服务器的流式批处理器
type PubBatcher struct {
	*pub.Batcher
	subject string
	conn    *nats.Conn

	metricFlush        *metrics.Counter
	metricBatchSizeMsg *metrics.Histogram
}

func NewPubBatcher(subject string, conn *nats.Conn) *PubBatcher {
	tb := &PubBatcher{
		subject: subject,
		conn:    conn,

		metricBatchSizeMsg: gmetrics.MetricBatchSizeMsg(subject),
		metricFlush:        gmetrics.MetricBatchFlush(subject),
	}

	tb.Batcher = pub.NewBatcher(func(data []byte, count int) {
		err := tb.conn.Publish(tb.subject, data)
		if err != nil {
			zap.L().Error("publish err", zap.Error(err))
		}

		tb.metricFlush.Add(count)
		tb.metricBatchSizeMsg.Update(float64(count))
	})

	return tb
}

func (bs *DataBus) NewIdx(serType pb.Server, serID uint8) *PubBatcher {
	return NewPubBatcher(idxSubjectName(serType, serID), bs.conn)
}

func (bs *DataBus) NewGroup(serType pb.Server) *PubBatcher {
	return NewPubBatcher(groupSubjectName(serType), bs.conn)
}

func (bs *DataBus) NewAll(serType pb.Server) *PubBatcher {
	return NewPubBatcher(allSubjectName(serType), bs.conn)
}
