package session

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"time"
)

//
// var recvBufPool = sync.Pool{
// 	New: func() any {
// 		// 分配一个足够大的缓冲区
// 		b := make([]byte, maxRecvMsgSize)
// 		return &b
// 	},
// }
//
// func (s *Session) readLoop2(cfg *Config) {
// 	defer func() {
// 		Close(s.in)
// 	}()
//
// 	head := make([]byte, 4)
// 	for {
// 		if cfg.ReadDeadline > 0 {
// 			_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
// 		}
//
// 		mt, r, err := s.conn.NextReader()
// 		if err != nil {
// 			zap.L().Warn("NextReader err", zap.Inline(s), zap.Error(err))
// 			return
// 		}
//
// 		if mt == websocket.CloseMessage {
// 			return
// 		} else if mt != websocket.BinaryMessage {
// 			continue
// 		}
//
// 		bufPtr := recvBufPool.Get().(*[]byte)
// 		buf := *bufPtr
//
// 		_, err = io.ReadFull(r, head)
// 		if err != nil {
// 			zap.L().Warn("read head err", zap.Inline(s), zap.Error(err))
// 			recvBufPool.Put(bufPtr)
// 			return
// 		}
// 		length := binary.BigEndian.Uint32(head)
// 		if length > maxRecvMsgSize {
// 			zap.L().Warn("msg len error", zap.Inline(s), zap.Error(err))
// 			recvBufPool.Put(bufPtr)
// 			return
// 		}
// 		buf = buf[:length]
// 		_, err = io.ReadFull(r, buf)
// 		if err != nil {
// 			zap.L().Warn("read data err", zap.Inline(s), zap.Error(err))
// 			recvBufPool.Put(bufPtr)
// 			return
// 		}
//
// 		// 异步处理，不能立即 Put 回池子。
// 		select {
// 		case s.in <- buf:
// 		case <-s.ctrl:
// 			return
// 		}
// 	}
// }

func (s *Session) readLoop(cfg *Config) {
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
		case <-s.ctrl:
			zap.L().Info("connection close by server", zap.Inline(s))
			return
		}
	}
}
