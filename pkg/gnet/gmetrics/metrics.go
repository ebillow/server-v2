package gmetrics

import (
	"fmt"
	"sync"

	"github.com/VictoriaMetrics/metrics"
)

// ======================= 发送侧指标 =======================
func MetricPubMsg(subject string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`msgq_pub_msgs_total{subject="%s"}`, subject))
}

func MetricPubDrop(subject string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`msgq_pub_drop_total{subject="%s"}`, subject))
}

func MetricBatchFlush(subject string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`msgq_batch_flush_total{subject="%s"}`, subject))
}

func MetricBatchSizeMsg(subject string) *metrics.Histogram {
	// VM 的 Histogram 自动管理 bucket，无需手动定义边界！
	return metrics.GetOrCreateHistogram(fmt.Sprintf(`msgq_batch_size_msgs{subject="%s"}`, subject))
}

func MetricNatsPubLatency(subject string) *metrics.Histogram {
	return metrics.GetOrCreateHistogram(fmt.Sprintf(`msgq_nats_publish_latency_ms{subject="%s"}`, subject))
}

// ======================= 接收侧指标 =======================
type SubMetrics struct {
	Subject     string
	DecodeErr   *metrics.Counter
	MsgCounters sync.Map // MsgID(uint32) -> *metrics.Counter
}

func NewSubMetrics(subject string) *SubMetrics {
	return &SubMetrics{
		Subject:   subject,
		DecodeErr: metrics.GetOrCreateCounter(fmt.Sprintf(`msgq_sub_decode_err_total{subject="%s"}`, subject)),
	}
}

func (sm *SubMetrics) GetMsgCounter(msgID uint32) *metrics.Counter {
	if v, ok := sm.MsgCounters.Load(msgID); ok {
		return v.(*metrics.Counter)
	}
	c := metrics.GetOrCreateCounter(fmt.Sprintf(`msgq_sub_msgs_total{subject="%s",msg_id="%d"}`, sm.Subject, msgID))
	actual, _ := sm.MsgCounters.LoadOrStore(msgID, c)
	return actual.(*metrics.Counter)
}

// 全局缓存
var (
	subMetricsMap    sync.Map // Subject(string) -> *SubMetrics
	handlerLatencies sync.Map // MsgID(uint32) -> *metrics.Histogram
)

func GetSubMetrics(subject string) *SubMetrics {
	if v, ok := subMetricsMap.Load(subject); ok {
		return v.(*SubMetrics)
	}
	sm := NewSubMetrics(subject)
	actual, _ := subMetricsMap.LoadOrStore(subject, sm)
	return actual.(*SubMetrics)
}

func GetHandlerLatencyMetric(msgID uint32) *metrics.Histogram {
	if v, ok := handlerLatencies.Load(msgID); ok {
		return v.(*metrics.Histogram)
	}
	h := metrics.GetOrCreateHistogram(fmt.Sprintf(`msgq_handler_latency_ms{msg_id="%d"}`, msgID))
	actual, _ := handlerLatencies.LoadOrStore(msgID, h)
	return actual.(*metrics.Histogram)
}

// ======================= 系统/连接层指标 =======================

func SetNatsConnStatus(server string, status float64) {
	// Gauge 直接 Set 值
	metrics.GetOrCreateGauge(fmt.Sprintf(`msgq_nats_conn_status{server="%s"}`, server), nil).Set(status)
}

func IncNatsReconnect(server string) {
	metrics.GetOrCreateCounter(fmt.Sprintf(`msgq_nats_reconnect_total{server="%s"}`, server)).Inc()
}

func IncSlowConsumer(subject string) {
	metrics.GetOrCreateCounter(fmt.Sprintf(`msgq_nats_slow_consumer_total{subject="%s"}`, subject)).Inc()
}
