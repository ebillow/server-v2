package v1

import (
	"context"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"time"
)

func (s *Session) readLoop(ctx context.Context, cfg *Config) {
	defer func() {
		close(s.in)
	}()

	for {
		if cfg.ReadDeadline > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
		}

		mt, data, err := s.conn.ReadMessage()
		if err != nil {
			zap.L().Warn("read message err", zap.Inline(s), zap.Error(err))
			return
		}
		if mt == websocket.CloseMessage {
			zap.L().Debug("connection close by client", zap.Inline(s))
			return
		} else if mt != websocket.BinaryMessage {
			continue
		}

		if len(data) > maxRecvMsgSize {
			zap.L().Warn("read msg too big", zap.Uint64("ses_id", s.Id), zap.Int("size", len(data)))
			return
		}

		select {
		case s.in <- data:
		case <-ctx.Done():
			zap.L().Info("connection close by server", zap.Inline(s))
			return
		}
	}
}
