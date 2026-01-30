package session

import (
	"github.com/gorilla/websocket"
	"server/pkg/logger"
	"time"
)

func (s *Session) readLoop(cfg *Config) {
	logger.Debugf("%s cli recv loop start", s.String())
	defer func() {
		close(s.in)
		logger.Debugf("%s cli recv loop stop", s.String())
	}()

	for {
		if cfg.ReadDeadline > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
		}
		mt, data, err := s.conn.ReadMessage()
		if err != nil {
			logger.Debugf("%d read message err:%v", s.Id, err)
			return
		}
		if mt == websocket.CloseMessage {
			logger.Debugf("%s connection close by client", s.String())
			return
		} else if mt != websocket.BinaryMessage {
			continue
		}

		select {
		case s.in <- data:
		case <-s.ctrl:
			logger.Debugf("%s connection close by server", s.String())
			return
		}
	}
}
