package session

import (
	"net"
	"net/http"
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
	shutDown  = make(chan struct{})
	waitGroup sync.WaitGroup
	netCfg    *Config
)

// StartWSServer 开始服务
func StartWSServer(listenEndPoint string, cfg *Config) {
	netCfg = cfg
	http.HandleFunc("/", handleClient)
	err := http.ListenAndServe(listenEndPoint, nil)
	if err != nil {
		zap.S().Errorf("listen err:%v", err)
		return
	}
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

// Close 关闭，并等待所有goroutine退出
func Close() {
	close(shutDown)
	waitGroup.Wait()
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
