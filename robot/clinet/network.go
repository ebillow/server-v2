package clinet

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net"
	"server/pkg/thread"
	"sync"
	"sync/atomic"
	"time"
)

var (
	shutDown  = make(chan struct{})
	waitGroup sync.WaitGroup
)

// Config 网络配置
type Config struct {
	ReadDeadline        time.Duration // time.Second * 1500
	OutChanSize         int           // 128
	ReadSocketBuffSize  int
	WriteSocketBuffSize int //
	RpmLimit            int
	RecvPkgLenLimit     uint32
	EvtChanSize         int
}

// StartTCPServer 开始tcp服务
func StartTCPServer(listenEndPoint string, cfg *Config) {
	// addr, err := net.ResolveTCPAddr("tcp4", listenEndPoint)
	// checkError(err)
	//
	// listener, err := net.ListenTCP("tcp", addr)
	// if err != nil {
	//	logger.Panic(err)
	// }
	// zap.S().Info("start listen on:", listener.Addr())
	//
	// for {
	//	conn, err := listener.AcceptTCP()
	//	if err != nil {
	//		logger.Warn("accept failed:", err)
	//		continue
	//	}
	//
	//	// set socket read buffer
	//	conn.SetReadBuffer(cfg.ReadSocketBuffSize)
	//	// // set socket write buffer
	//	conn.SetWriteBuffer(cfg.WriteSocketBuffSize)
	//
	//	go handleClient(conn, cfg)
	// }
}

func handleClient(conn *websocket.Conn, cfg *Config) {
	defer func() {
		if err := recover(); err != nil {
			thread.PrintStack(err)
		}
	}()

	var s Session
	s.conn = conn
	defer s.conn.Close()

	s.in = make(chan []byte) // no active_role
	defer func() {
		close(s.in)
		// zap.S().Debug("recv loop stop")
	}()

	host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		zap.S().Error("cannot get remote address:", err)
		return
	}
	s.Ip = net.ParseIP(host)
	zap.S().Debugf("new connection from:%v port:%v", host, port)

	s.ctrl = make(chan struct{})

	s.start(cfg)
}

// Close 关闭，并等待所有goroutine退出
func Close() {
	close(shutDown)
	zap.S().Debugf("start wait %v", waitGroup)
	waitGroup.Wait()
}

/************************************************************/
var (
	CliSess sync.Map
	CliCnt  int32

	// cliConnCnt         map[string]int
	// blackList			sync.Map
	// cliConnChan			chan string
)

/**************************************************************/
func AddCliSession(cliSesID uint32, c *Session) {
	// ip := c.Ip.String()
	// if _, ok := blackList.Load(ip); ok{
	//	c.Close()
	//	return
	// }
	// cliConnChan <- ip
	CliSess.Store(cliSesID, c)
	atomic.AddInt32(&CliCnt, 1)
}

func RemoveCliSession(cliSesID uint32) {
	CliSess.Delete(cliSesID)
	atomic.AddInt32(&CliCnt, -1)
}
