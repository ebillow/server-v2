package ghttp

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"os"
	"server/pkg/logger"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	logger.NewZapLog("../../bin/logger/test.logger", logger.Config{
		Level:   0,
		Console: true,
	})
	trace.Store(true)
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	Start(ctx, wg, 8080)
	EG().POST("/user/authenticate", func(c *gin.Context) {
		c.JSON(200, gin.H{"test": "test"})
	})
	time.Sleep(time.Second)

	err := req(ctx)
	require.NoError(t, err)

	cancel()
	wg.Wait()
}

func req(ctx context.Context) error {
	cli := NewClient("").SetBaseURL("http://127.0.0.1:8080")
	r := cli.R().SetHeader("Content-Type", "application/json").SetContext(ctx)

	// 1. authenticate
	{
		resp, err := r.SetBody(map[string]any{"userToken": 1111}).Post("/user/authenticate")
		if err != nil {
			return err
		}
		if resp.StatusCode() != 200 {
			return fmt.Errorf("status code: %d", resp.StatusCode())
		}
	}
	return nil
}
