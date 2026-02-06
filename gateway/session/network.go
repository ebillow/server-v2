package session

import (
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"server/pkg/logger"
	"server/pkg/thread"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxMsgCount = 655350
)

// Config 网络配置
type Config struct {
	ReadDeadline        time.Duration // time.Second * 1500
	OutChanSize         int           // 128
	ReadSocketBuffSize  int
	WriteSocketBuffSize int //
	RpmLimit            int
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
		logger.Errorf("listen err:%v", err)
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
		logger.Debug("connect err: upgrade:", err)
		return
	}

	host, port, err := net.SplitHostPort(c.RemoteAddr().String())
	if err != nil {
		logger.Error("cannot get remote address:", err)
		return
	}
	c.SetReadLimit(int64(netCfg.RecvPkgLenLimit))
	var s = &Session{}
	s.conn = c
	s.Ip = net.ParseIP(host).String()
	logger.Tracef("new connection from:%v port:%v", host, port)

	s.start()
}

/*
func handleClient(conn net.Conn, cfg *Config) {
	defer func() {
		if err := recover(); err != nil {
			util.PrintStack(err)
		}
	}()

	var s = &Session{}
	s.conn = conn
	defer s.conn.Close()

	s.in = make(chan []byte) //no active_role
	defer func() {
		close(s.in)
		//logger.Debug("recv loop stop")
	}()

	host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		logger.Error("cannot get remote address:", err)
		return
	}
	s.Ip = net.ParseIP(host)
	logger.Tracef("new connection from:%v port:%v", host, port)

	s.ctrl = make(chan struct{})

	s.start(conn, cfg)
}
*/

// Close 关闭，并等待所有goroutine退出
func Close() {
	close(shutDown)
	waitGroup.Wait()
}

/************************************************************/
var (
	CliSess sync.Map
	CliCnt  int32
)

/**************************************************************/
func AddCliSession(cliSesID uint64, c *Session) {
	CliSess.Store(cliSesID, c)
	atomic.AddInt32(&CliCnt, 1)
}

func RemoveCliSession(cliSesID uint64) {
	CliSess.Delete(cliSesID)
	atomic.AddInt32(&CliCnt, -1)
}

func GetCliSessCnt() int32 {
	return atomic.LoadInt32(&CliCnt)
}

func GetCliSession(cliSesID uint64) *Session {
	s, ok := CliSess.Load(cliSesID)
	if ok {
		return s.(*Session)
	}
	return nil
}

// **********************************
var isTraceProto int32

func IsTraceProto() bool {
	return atomic.LoadInt32(&isTraceProto) == 1
}

func SetTraceProto(v bool) {
	logger.Infof("set trace msg :%t", v)
	if v {
		atomic.StoreInt32(&isTraceProto, 1)
	} else {
		atomic.StoreInt32(&isTraceProto, 0)
	}
}
