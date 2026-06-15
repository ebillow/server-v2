package session

import (
	"context"
	"net"
	"net/http"
	"server/api/pb"
	"server/pkg/thread"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Config 网络配置
type Config struct {
	ReadDeadline        time.Duration // time.Second * 1500
	OutChanSize         int           //
	ReadSocketBuffSize  int
	WriteSocketBuffSize int //
	RecvPkgLenLimit     uint32
}

var (
	waitGroup sync.WaitGroup
	netCfg    *Config

	httpSrv *http.Server
)

// StartWSServer 开始服务
func StartWSServer(listenEndPoint string, cfg *Config) {
	netCfg = cfg
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleClient)

	httpSrv = &http.Server{
		Addr:    listenEndPoint,
		Handler: mux,
	}

	thread.GoSafe(func() {
		zap.S().Infof("WS Server starting at %s", listenEndPoint)
		// 当 Shutdown 被调用时，ListenAndServe 会返回 http.ErrServerClosed
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Fatalf("listen err: %v", err)
		}
	})
}

func handleClient(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			thread.PrintStack(err)
		}
	}()

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  netCfg.ReadSocketBuffSize,
		WriteBufferSize: netCfg.WriteSocketBuffSize,
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		zap.S().Debug("connect err: upgrade:", err)
		return
	}

	host, port, err := net.SplitHostPort(c.RemoteAddr().String())
	if err != nil {
		zap.S().Error("cannot get remote address:", err)
		return
	}
	c.SetReadLimit(int64(netCfg.RecvPkgLenLimit))
	var s = &Session{}
	s.conn = c
	s.Ip = net.ParseIP(host).String()
	zap.S().Debugf("new connection from:%v port:%v", host, port)

	s.start()
}

// GracefulStop 关闭，并等待所有goroutine退出
func GracefulStop() {
	zap.S().Info("Shutting down WS Server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		zap.S().Errorf("HTTP Server Shutdown Error: %v", err)
	}

	zap.S().Info("Kicking all connected clients...")
	for _, shard := range shards {
		shard.mtx.RLock() //  s.Close() 内部会去调 Remove，Remove 要拿写锁
		// 为了防止 s.Close() 时发生死锁 (读锁未释放又去申请写锁)，先克隆一份 session 列表
		clientsToClose := make([]*Session, 0, len(shard.data))
		for _, s := range shard.data {
			clientsToClose = append(clientsToClose, s)
		}
		shard.mtx.RUnlock()

		// 在无锁状态下执行踢人逻辑
		for _, s := range clientsToClose {
			s.Close(pb.DisconnectReason_ShutDown)
		}
	}
	waitGroup.Wait()
	zap.S().Info("WS Server gracefully stopped.")
}

/************************************************************/

const shardCount = 64

var (
	shards = make([]*roleShard, shardCount)
)

type roleShard struct {
	mtx  sync.RWMutex
	data map[uint64]*Session
}

func init() {
	for i := 0; i < shardCount; i++ {
		shards[i] = &roleShard{
			data: make(map[uint64]*Session),
		}
	}
}
func getRoleShard(sesID uint64) *roleShard {
	return shards[sesID&(shardCount-1)]
}

func Add(sesID uint64, data *Session) {
	rs := getRoleShard(sesID)
	rs.mtx.Lock()
	rs.data[sesID] = data
	rs.mtx.Unlock()
}

func Remove(sesID uint64) {
	rs := getRoleShard(sesID)
	rs.mtx.Lock()
	delete(rs.data, sesID)
	rs.mtx.Unlock()
}

func Count() int {
	var count int
	for _, shard := range shards {
		shard.mtx.RLock()
		count += len(shard.data)
		shard.mtx.RUnlock()
	}
	return count
}

func Get(sesID uint64) *Session {
	rs := getRoleShard(sesID)
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()

	return rs.data[sesID]
}
