package monitor

import (
	"time"

	"github.com/VictoriaMetrics/metrics"
)

var (
	// 定义一个直方图，用于统计耗时（自动包含 P50, P90, P95, P99）
	// {api="login"} 是标签(Label)，方便在 Grafana 里区分不同接口
	costHistogram = metrics.NewHistogram(`request_duration_seconds{api="my_api"}`)

	// 定义一个计数器，用于统计请求总数和 QPS
	reqTotal = metrics.NewCounter(`request_total{api="my_api"}`)

	// 定义一个计数器，用于统计失败数
	errTotal = metrics.NewCounter(`request_error_total{api="my_api"}`)
)

// Add 记录耗时
func Add(cost time.Duration) {
	reqTotal.Inc()
	// 统一转换为秒进行记录
	costHistogram.Update(cost.Seconds())
}

// Fail 记录失败
func Fail() {
	errTotal.Inc()
}

//
// // Register 启动监控接口
// func Register() {
// 	ghttp.EG().GET("/metrics", func(context *gin.Context) {
// 		metrics.WritePrometheus(context.Writer, true)
// 	})
// }
