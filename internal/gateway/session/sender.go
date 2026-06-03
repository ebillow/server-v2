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
	"google.golang.org/protobuf/proto"
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
func (s *Session) Send(msg proto.Message) bool {
	msgID, ok := pb.GetMsgIDS2C(msg)
	if !ok {
		zap.L().Error("send msg error, msg id not exists", zap.String("type", fmt.Sprintf("%T", msg)))
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

// sendLoop todo 合并发送，应对波峰
func (s *Session) sendLoop(ctx context.Context) {
	defer func() {
		s.Close(pb.DisconnectReason_NetErr)
		waitGroup.Done()
	}()

	var headerBuf [4]byte
	for {
		select {
		case <-s.out.Sig():
			_ = s.conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
			s.out.Range(func(v MsgSend) {
				w, err := s.conn.NextWriter(websocket.BinaryMessage)
				if err != nil {
					zap.L().Warn("NextWriter error", zap.Error(err))
					s.Close(pb.DisconnectReason_NetErr)
				}

				binary.BigEndian.PutUint32(headerBuf[0:4], v.ID)

				_, err = w.Write(headerBuf[:])
				if err == nil && len(v.Data) > 0 {
					_, err = w.Write(v.Data)
				}
				// 立刻关闭 Writer 刷入网络
				_ = w.Close()

				if err != nil {
					return
				}
			})

		case <-ctx.Done():
			return
		}
	}
}
