package clinet

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net"
	"net/url"
)

// DailTCP 连接服务器，
func DailWebsocket(addr string, cfg *Config) (*Session, error) {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	// zap.S().Debugf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		zap.S().Error("dial:", err)
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
			zap.S().Debug("recv loop stop")
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
	}()
	return s, nil
}
