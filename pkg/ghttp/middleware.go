package ghttp

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io"
	"server/pkg/thread"
	"sync"
	"sync/atomic"
	"time"
)

var trace atomic.Bool

const (
	// MaxLogSize 限制日志记录的最大 Body 长度 (例如 4KB)
	// 防止大体积 Payload 导致 OOM 和日志系统崩溃
	MaxLogSize = 4096
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, MaxLogSize))
	},
}

// limitRespWriter 带有容量限制的响应截获器
type limitRespWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	loggedSize int
}

func (w *limitRespWriter) Write(b []byte) (int, error) {
	// 只截获前 MaxLogSize 个字节用于日志记录
	if w.loggedSize < MaxLogSize {
		writeLen := len(b)
		if w.loggedSize+writeLen > MaxLogSize {
			writeLen = MaxLogSize - w.loggedSize
		}
		w.body.Write(b[:writeLen])
		w.loggedSize += writeLen
	}
	// 正常向客户端写入数据
	return w.ResponseWriter.Write(b)
}

func logMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ua := c.Request.UserAgent()
		if ua == "healthProbe" {
			c.Next()
			return
		}

		startTime := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		var reqData any
		contentType := c.ContentType()

		// 1. 安全读取 Request Body (避免大包直接撑爆内存)
		switch contentType {
		case "application/json":
			// 多读 1 个字节，用于判断是否超长
			limitReader := io.LimitReader(c.Request.Body, MaxLogSize+1)
			reqBytes, _ := io.ReadAll(limitReader)

			if len(reqBytes) > MaxLogSize {
				// 说明 Body 超过了限制长度，截断用于日志记录
				reqData = string(reqBytes[:MaxLogSize]) + " ...(truncated)"
				// 核心逻辑：将读出来的部分和没读完的部分重新拼接，还给 Gin
				// 这样后续的 c.ShouldBindJSON 依然能正常读取完整的结构
				c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(reqBytes), c.Request.Body))
			} else {
				// 没有超长，正常记录，并将完整读取的 bytes 放回 Body
				reqData = string(reqBytes)
				c.Request.Body = io.NopCloser(bytes.NewReader(reqBytes))
			}

		case "application/x-www-form-urlencoded":
			_ = c.Request.ParseForm()
			reqData = c.Request.PostForm

		case "multipart/form-data":
			// 文件上传绝对不能读 body 到内存日志里，只记录一个标识位
			reqData = "[multipart/form-data skipped]"
		}

		// 2. 使用 sync.Pool 获取响应 Buffer
		respBuf := bufferPool.Get().(*bytes.Buffer)
		respBuf.Reset()
		defer bufferPool.Put(respBuf)

		respWriter := &limitRespWriter{
			ResponseWriter: c.Writer,
			body:           respBuf,
		}
		c.Writer = respWriter

		// 3. 放行请求
		c.Next()

		// 4. 计算与日志记录
		latency := time.Since(startTime)
		status := c.Writer.Status()

		lv := zap.InfoLevel
		if status >= 500 {
			lv = zap.ErrorLevel
		} else if status >= 400 {
			lv = zap.WarnLevel
		}

		if lv > zap.InfoLevel || trace.Load() {
			respStr := respBuf.String()
			if respWriter.loggedSize >= MaxLogSize {
				respStr += " ...(truncated)"
			}

			zap.S().With(
				zap.Any("req", reqData),
				zap.String("resp", respStr), // 换成 String，防止截断的 JSON 破坏日志收集器的解析
				zap.String("remote", c.ClientIP()),
				zap.String("contentType", contentType),
				zap.String("ua", ua),
				zap.Duration("latency", latency),
				zap.Int("status", status),
			).Logf(lv, "http-msg | [%d] %s %s", status, c.Request.Method, path)
		}
	}
}

func recoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		panics := thread.RunSafe(func() { c.Next() })
		if panics {
			Fail(c, 500, 0, "")
			c.Abort()
		}
	}
}
