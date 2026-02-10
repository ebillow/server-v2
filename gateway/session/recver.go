package session

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"time"
)

func (s *Session) readLoop(cfg *Config) {
	zap.S().Debugf("%s cli recv loop start", s.String())
	defer func() {
		close(s.in)
		zap.S().Debugf("%s cli recv loop stop", s.String())
	}()

	for {
		if cfg.ReadDeadline > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
		}
		mt, data, err := s.conn.ReadMessage()
		if err != nil {
			zap.S().Debugf("%d read message err:%v", s.Id, err)
			return
		}
		if mt == websocket.CloseMessage {
			zap.S().Debugf("%s connection close by client", s.String())
			return
		} else if mt != websocket.BinaryMessage {
			continue
		}

		select {
		case s.in <- data:
		case <-s.ctrl:
			zap.S().Debugf("%s connection close by server", s.String())
			return
		}
	}
}
