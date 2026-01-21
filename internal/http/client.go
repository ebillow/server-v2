package http

import (
	"crypto/tls"
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"net/http"
)

func NewClient(proxy string) *resty.Client {
	client := resty.New()
	client.SetTransport(&http.Transport{
		MaxIdleConnsPerHost: 100,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	})
	if proxy != "" {
		client.SetProxy(proxy)
	}
	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		if trace.Load() {
			req := resp.Request
			reqBytes, _ := json.Marshal(req.Body)
			zap.S().WithOptions(zap.AddCallerSkip(4)).With(
				zap.Int("httpCode", resp.StatusCode()),
				zap.Any("req", json.RawMessage(reqBytes)),
				zap.Any("resp", json.RawMessage(resp.Body())),
				zap.Duration("latency", resp.Time()),
			).Infof("http-cli trip | %s %s", req.Method, req.URL)
		}
		return nil
	})
	return client
}
