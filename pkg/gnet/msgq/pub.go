package msgq

import (
	"server/api/pb"
	"server/pkg/gnet/batcher"
	"server/pkg/gnet/dep"
	"server/pkg/gnet/gctx"
	"server/pkg/gnet/gmetrics"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// 发送函数 会接管 data 的生命周期，调用结束后请勿修改 data

// Send 指定发送
func (bs *DataBus) Send(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}
	pbt, err := bs.getIdxPubBatcher(serType, serID)
	if err != nil {
		return err
	}
	return pbt.Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSerID: bs.serID,
		FromSer:   bs.serType,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
	})
}

func (bs *DataBus) ForwardToRole(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}

	pbt, err := bs.getIdxPubBatcher(serType, serID)
	if err != nil {
		return err
	}
	return pbt.Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSerID: bs.serID,
		FromSer:   bs.serType,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
		Flag:      gctx.Forward,
	})
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}

	pbt, err := bs.getGroupPubBatcher(serType)
	if err != nil {
		return err
	}

	return pbt.Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ActorID:   actorID,
		SesID:     sesID,
	})
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serType pb.Server, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}

	pbt, err := bs.getAllPubBatcher(serType)
	if err != nil {
		return err
	}

	return pbt.Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ActorID:   actorID,
		SesID:     sesID,
	})
}

// Relay 转发给指定服
func (bs *DataBus) Relay(serType pb.Server, serID uint8, msgID uint32, data []byte, actorID uint64, sesID uint64) error {
	if bs.closed.Load() {
		return dep.ErrClosed
	}

	pbt, err := bs.getGroupPubBatcher(pb.Server_Center)
	if err != nil {
		return err
	}

	return pbt.Add(gctx.Context{
		MsgID:     msgID,
		Data:      data,
		FromSer:   bs.serType,
		FromSerID: bs.serID,
		ToSer:     uint8(serType),
		ToSerID:   serID,
		ActorID:   actorID,
		SesID:     sesID,
		Flag:      gctx.Forward,
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

func (bs *DataBus) getIdxPubBatcher(serType pb.Server, serID uint8) (*PubBatcher, error) {
	if serType >= SvcTypeMax || serID >= SvcIDMax {
		return nil, dep.ErrArg
	}
	tb := bs.pubIDXs[serType][serID].Load()
	if tb != nil {
		return tb, nil
	}

	bs.pubIDXMtx.Lock()
	defer bs.pubIDXMtx.Unlock()

	if tb = bs.pubIDXs[serType][serID].Load(); tb != nil {
		return tb, nil
	}

	subject := idxSubjectName(serType, serID)
	tb = NewPubBatcher(subject, bs.conn)
	bs.pubIDXs[serType][serID].Store(tb)

	return tb, nil
}

func (bs *DataBus) getGroupPubBatcher(serType pb.Server) (*PubBatcher, error) {
	if serType >= SvcTypeMax {
		return nil, dep.ErrArg
	}
	tb := bs.pubGroup[serType].Load()
	if tb != nil {
		return tb, nil
	}

	bs.pubGroupMtx.Lock()
	defer bs.pubGroupMtx.Unlock()

	if tb = bs.pubGroup[serType].Load(); tb != nil {
		return tb, nil
	}

	subject := groupSubjectName(serType)
	tb = NewPubBatcher(subject, bs.conn)
	bs.pubGroup[serType].Store(tb)

	return tb, nil
}

func (bs *DataBus) getAllPubBatcher(serType pb.Server) (*PubBatcher, error) {
	if serType >= SvcTypeMax {
		return nil, dep.ErrArg
	}

	tb := bs.pubAll[serType].Load()
	if tb != nil {
		return tb, nil
	}

	bs.pubAllMtx.Lock()
	defer bs.pubAllMtx.Unlock()

	if tb = bs.pubAll[serType].Load(); tb != nil {
		return tb, nil
	}

	subject := allSubjectName(serType)
	tb = NewPubBatcher(subject, bs.conn)
	bs.pubAll[serType].Store(tb)

	return tb, nil
}

func (bs *DataBus) flushAllBatchers() {
	for i := range bs.pubIDXs {
		for j := range bs.pubIDXs[i] {
			if tb := bs.pubIDXs[i][j].Load(); tb != nil {
				tb.StopAndFlush()
			}
		}
	}

	for i := range bs.pubGroup {
		if tb := bs.pubGroup[i].Load(); tb != nil {
			tb.StopAndFlush()
		}
	}

	for i := range bs.pubAll {
		if tb := bs.pubAll[i].Load(); tb != nil {
			tb.StopAndFlush()
		}
	}
}

// PubBatcher 针对单个目标服务器的流式批处理器
type PubBatcher struct {
	*batcher.BaseBatcher
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

	tb.BaseBatcher = batcher.NewBaseBatcher(func(data []byte, count int) {
		err := tb.conn.Publish(tb.subject, data)
		if err != nil {
			zap.L().Error("publish err", zap.Error(err))
		}

		tb.metricFlush.Add(count)
		tb.metricBatchSizeMsg.Update(float64(count))
	})

	return tb
}
