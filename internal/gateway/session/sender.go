package session

import (
	"context"
	"encoding/binary"
	"fmt"
	pb "server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/gnet/trace"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// SendBytes 发送数据给客户端
func (s *Session) SendBytes(msgID uint32, data []byte) {
	err := s.out.Push(MsgSend{
		ID:   msgID,
		Data: data,
	})
	if err != nil {
		zap.L().Warn("send to client err", zap.Uint32("msgID", msgID), zap.Error(err))
	} else {
		if trace.Rule.ShouldLog(msgID, 0, s.Id) {
			zap.L().Info(">>> to client: "+msgid.MsgIDS2C_name[int32(msgID)],
				zap.Uint32("msgID", msgID),
				zap.Inline(s),
			)
		}
	}
}

// Send 发送proto数据给客户端
func (s *Session) Send(msg pb.VTMessage) bool {
	msgID, ok := pb.GetMsgIDS2C(msg)
	if !ok {
		zap.L().Error("send msg error, msg id not exists", zap.String("type", fmt.Sprintf("%T", msg)))
		return false
	}
	return s.SendPB(msgid.MsgIDS2C(msgID), msg)
}

// SendPB 发送proto数据给客户端
func (s *Session) SendPB(msgID msgid.MsgIDS2C, msg pb.VTMessage) bool {
	if msg == nil {
		zap.S().Warnf("msg is nil")
		return false
	}

	var b []byte
	var err error
	b, err = msg.MarshalVT()
	if err != nil {
		zap.S().Warnf("send pb, marshal error:%v", err)
		return false
	}

	s.SendBytes(uint32(msgID), b)
	return true
}

// sendLoop
func (s *Session) sendLoop(ctx context.Context) {
	defer func() {
		s.onLoopExit()
	}()

	var headerBuf [4]byte
	for {
		select {
		case <-s.out.Sig():
			if err := s.drainAndWrite(headerBuf[:]); err != nil {
				s.Close(pb.DisconnectReason_NetErr)
				return
			}
		case <-ctx.Done():
			_ = s.drainAndWrite(headerBuf[:])
			s.writeCloseFrame()
			return
		}
	}
}

// todo 合并发送，应对波峰
// drainAndWrite 把队列中所有消息写到 conn
func (s *Session) drainAndWrite(headerBuf []byte) error {
	var firstErr error
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	s.out.Range(func(v MsgSend) {
		if firstErr != nil {
			return
		}

		w, err := s.conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			firstErr = err
			return
		}

		binary.BigEndian.PutUint32(headerBuf[0:4], v.ID)
		_, err = w.Write(headerBuf[:4])
		if err == nil && len(v.Data) > 0 {
			_, err = w.Write(v.Data)
		}
		if closeErr := w.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			firstErr = err
		}
	})

	if firstErr != nil {
		zap.L().Warn("write to client error",
			zap.Error(firstErr), zap.Inline(s))
	}
	return firstErr
}

// writeCloseFrame 发送 WebSocket Close 帧
func (s *Session) writeCloseFrame() {
	_ = s.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	_ = s.conn.WriteMessage(websocket.CloseMessage, msg)
}
