package v1

import (
	"context"
	"encoding/binary"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/gnet/trace"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

// SendBytes 发送数据给客户端 todo 是否改成队列
func (s *Session) SendBytes(msgID uint32, data []byte) {
	s.out <- MsgSend{ID: msgID, Data: data}
	if trace.Rule.ShouldLog(msgID, 0, s.Id) {
		zap.L().Info(">>> to client: "+msgid.MsgIDS2C_name[int32(msgID)],
			zap.Uint32("msgID", msgID),
			zap.Inline(s),
			logger.Magenta.Field(),
		)
	}
}

// Send 发送proto数据给客户端
func (s *Session) Send(msg proto.Message) bool {
	msgID, err := pb.GetMsgIDS2C(msg)
	if err != nil {
		zap.L().Warn("send error", zap.Error(err), zap.Inline(s))
		return false
	}
	return s.SendPB(msgid.MsgIDS2C(msgID), msg)
}

// SendPB 发送proto数据给客户端
func (s *Session) SendPB(msgID msgid.MsgIDS2C, msg proto.Message) bool {
	if msg == nil {
		zap.S().Warnf("msg is nil")
		return false
	}

	var b []byte
	var err error
	b, err = proto.Marshal(msg)
	if err != nil {
		zap.S().Warnf("send pb, marshal error:%v", err)
		return false
	}

	s.SendBytes(uint32(msgID), b)
	return true
}

// 看需求，可以合并发送
func (s *Session) sendLoop(ctx context.Context) {
	var headerBuf [4]byte

	for {
		select {
		case p := <-s.out:
			w, err := s.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				zap.L().Warn("NextWriter error", zap.Error(err))
				return
			}

			binary.BigEndian.PutUint32(headerBuf[0:4], p.ID)

			_, err = w.Write(headerBuf[:])
			if err != nil {
				return
			}

			if len(p.Data) > 0 {
				_, err = w.Write(p.Data)
				if err != nil {
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}
