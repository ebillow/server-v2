package ghttp

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io"
	"server/pkg/thread"
	"sync/atomic"
	"time"
)

var trace atomic.Bool

type respBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *respBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
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
		method := c.Request.Method

		var reqBytes []byte
		var contentType = c.ContentType()
		switch contentType {
		case "application/json":
			reqBytes, _ = io.ReadAll(c.Request.Body)
			_ = c.Request.Body.Close()
			c.Request.Body = io.NopCloser(bytes.NewReader(reqBytes)) // reset request body
		case "application/x-www-form-urlencoded", "multipart/form-data":
			_ = c.Request.ParseForm()
			postBody := make(map[string]any)
			for k, v := range c.Request.PostForm {
				if len(v) == 1 {
					postBody[k] = v[0]
				} else {
					postBody[k] = v
				}
			}
			reqBytes, _ = json.Marshal(postBody)
		}

		respWriter := &respBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = respWriter // override writer

		remoteIP := c.ClientIP()

		c.Next()

		latency := time.Since(startTime)
		status := c.Writer.Status()

		var lv = zap.ErrorLevel
		if status >= 500 {
			lv = zap.ErrorLevel
		} else if status >= 400 {
			lv = zap.WarnLevel
		} else if status >= 200 {
			lv = zap.InfoLevel
		}

		if lv > zap.InfoLevel || trace.Load() {
			zap.S().With(
				zap.Any("req", json.RawMessage(reqBytes)),
				zap.Any("resp", json.RawMessage(respWriter.body.Bytes())),
				zap.String("remote", remoteIP),
				zap.String("contentType", contentType),
				zap.String("ua", ua),
				zap.Duration("latency", latency),
				zap.Int("status", status),
			).Logf(lv, "http-msg | [%d] %s %s", status, method, path)
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
