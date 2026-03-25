package v2

import (
	"context"
	"encoding/binary"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/gnet/codec"
	"server/pkg/gnet/trace"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"time"
)

// SendBytes 发送数据给客户端
func (s *Session) SendBytes(msgID uint32, data []byte, msg *pb.NatsMsg) {
	err := s.out.Push(MsgSend{
		ID:   msgID,
		Data: data,
		Msg:  msg,
	})
	if err != nil {
		zap.L().Warn("send to client err", zap.Uint32("msgID", msgID), zap.Error(err))
	} else {
		if trace.Rule.ShouldLog(msgID, 0, s.Id) {
			zap.L().Info(">>> to client: "+msgid.MsgIDS2C_name[int32(msgID)],
				zap.Uint32("msgID", msgID),
				zap.Inline(s),
				logger.Magenta.Field(),
			)
		}
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

	s.SendBytes(uint32(msgID), b, nil)
	return true
}

// 看需求，可以合并发送
func (s *Session) sendLoop(ctx context.Context) {
	defer func() {
		s.Close(pb.DisconnectReason_NetErr)
		waitGroup.Done()
	}()

	var headerBuf [4]byte
	for {
		select {
		case <-s.out.Sig():
			s.out.Range(func(v MsgSend) bool {
				_ = s.conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
				w, err := s.conn.NextWriter(websocket.BinaryMessage)
				if err != nil {
					zap.L().Warn("NextWriter error", zap.Error(err))
					if v.Msg != nil {
						codec.PutNatsMsg(v.Msg)
					}
					s.Close(pb.DisconnectReason_NetErr)
					return false // 发生致命网络错误，退出循环并断开
				}

				binary.BigEndian.PutUint32(headerBuf[0:4], v.ID)

				_, err = w.Write(headerBuf[:])
				if err == nil && len(v.Data) > 0 {
					_, err = w.Write(v.Data)
				}
				// 立刻关闭 Writer 刷入网络
				_ = w.Close()

				if v.Msg != nil {
					codec.PutNatsMsg(v.Msg)
				}
				if err != nil {
					return false
				}

				return true
			})

		case <-ctx.Done():
			return
		}
	}
}
