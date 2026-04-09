package v2

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var recvBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, maxRecvMsgSize)
		return &b
	},
}

func (s *Session) readLoop(ctx context.Context, cfg *Config) {
	defer func() {
		s.Close(pb.DisconnectReason_Normal)
		waitGroup.Done()
	}()

	for {
		if cfg.ReadDeadline > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
		}
		mt, r, err := s.conn.NextReader()
		if err != nil {
			zap.L().Warn("NextReader err", zap.Inline(s), zap.Error(err))
			return
		}

		if mt == websocket.CloseMessage {
			return
		} else if mt != websocket.BinaryMessage {
			continue
		}

		bufPtr := recvBufPool.Get().(*[]byte)
		data := *bufPtr
		var total int

		for {
			n, err := r.Read(data[total:])
			total += n

			if err == io.EOF {
				break // 当前帧读取完毕
			}
			if err != nil {
				zap.L().Warn("read payload err", zap.Inline(s), zap.Error(err))
				recvBufPool.Put(bufPtr)
				return
			}
			if total == len(data) {
				zap.L().Warn("message length >= MaxMsgSize", zap.Inline(s), zap.Int("limit", len(data)))
				recvBufPool.Put(bufPtr)
				return
			}
		}

		msgData := data[:total]
		s.forwardToSrv(msgData)

		recvBufPool.Put(bufPtr)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func Decode(src []byte) (msgID uint32, data []byte, err error) {
	dataLen := len(src)
	const minDataLen = 4
	if dataLen < minDataLen {
		return 0, nil, errors.New("packet head < 6")
	}

	msgID = binary.BigEndian.Uint32(src[0:4])
	data = src[4:]
	return
}

func (s *Session) forwardToSrv(src []byte) {
	msgID, data, err := Decode(src)
	if err != nil {
		zap.L().Warn("read packet err", zap.Inline(s), zap.Error(err))
		s.Close(pb.DisconnectReason_DecodeErr)
		return
	}

	serType := pb.Server(msgID / 100000)
	serID := s.getSerID(serType)
	err = msgq.Q.Send(serType, serID, msgID, data, 0, s.Id) // nats Publish很快
	if err != nil {
		zap.L().Info(">>> to server: "+msgid.MsgIDC2S_name[int32(msgID)],
			zap.Uint32("msgID", msgID),
			zap.String("to", serType.String()),
			zap.Int32("idx", serID),
			zap.Inline(s),
		)
	}
	if trace.Rule.ShouldLog(msgID, 0, s.Id) {
		zap.L().Info(">>> to server: "+msgid.MsgIDC2S_name[int32(msgID)],
			zap.Uint32("msgID", msgID),
			zap.String("to", serType.String()),
			zap.Int32("idx", serID),
			zap.Inline(s),
		)
	}
}
