package ghttp

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"server/pkg/thread"
	"server/pkg/util"
	"sync"
	"time"
)

var (
	eg *gin.Engine
)

func EG() *gin.Engine {
	return eg
}

func Start(ctx context.Context, wait *sync.WaitGroup, port int) {
	eg = NewEngine()
	thread.GoSafe(func() {
		Serve(ctx, wait, eg, port)
	})
}

// NewEngine 创建 gin.Engine
func NewEngine() *gin.Engine {
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.RemoveExtraSlash = true
	corsConfig := cors.New(cors.Config{
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Access-Token"},
		MaxAge:          12 * time.Hour,
		AllowAllOrigins: true,
	})

	r.Use(logMiddleware(), corsConfig)
	if !util.Debug {
		r.Use(recoverMiddleware(), corsConfig)
		gin.SetMode(gin.ReleaseMode)
	}

	r.NoMethod(func(c *gin.Context) { Fail(c, http.StatusMethodNotAllowed, 400, "method not allowed") })
	r.NoRoute(func(c *gin.Context) { Fail(c, http.StatusNotFound, 400, "api not found") })

	r.Any("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, &Response{
			Code: 0,
			Data: "replay",
		})
	})

	// 注入 pprof; 访问 http//host:port/debug/pprof
	pprof.Register(r)

	return r
}

func Serve(ctx context.Context, wait *sync.WaitGroup, r *gin.Engine, port int) {
	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: r,
	}

	wait.Add(1)
	go func() {
		select {
		case <-ctx.Done():
			err := srv.Shutdown(context.Background())
			if err != nil {
				zap.S().Warnf("http server shutdown error: %s", err.Error())
			}
			wait.Done()
		}
	}()

	zap.S().Infof("http listen at: %d", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		zap.S().Errorf("http listen err: %v", err)
	}
}
