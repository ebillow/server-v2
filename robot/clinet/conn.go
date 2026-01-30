package clinet

import (
	"github.com/gorilla/websocket"
	"net"
	"net/url"
	"server/pkg/logger"
)

// DailTCP 连接服务器，
func DailWebsocket(addr string, cfg *Config) (*Session, error) {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	// logger.Debugf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logger.Error("dial:", err)
		return nil, err
	}

	return handleCliConn(c, cfg)
}

func handleCliConn(conn *websocket.Conn, cfg *Config) (*Session, error) {
	s := &Session{}
	s.conn = conn
	go func() {
		defer s.conn.Close()

		s.in = make(chan []byte) // no active_role
		defer func() {
			close(s.in)
			logger.Debug("recv loop stop")
		}()

		host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			logger.Error("cannot get remote address:", err)
			return
		}
		s.Ip = net.ParseIP(host)
		logger.Tracef("new connection from:%v port:%v", host, port)

		s.ctrl = make(chan struct{})

		s.start(cfg)
	}()
	return s, nil
}
