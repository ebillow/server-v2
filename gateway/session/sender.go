package session

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/gnet/trace"
	"server/pkg/logger"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
)

// SendBytes 发送数据给客户端
func (s *Session) SendBytes(msgID uint32, data []byte) {
	s.out <- &MsgSend{ID: msgID, Data: data}

	if trace.Rule.ShouldLog(msgID, 0, s.Id) {
		zap.L().Info(">>> to client: "+msgid.MsgIDS2C_name[int32(msgID)],
			zap.Uint32("msgID", msgID),
			zap.Uint64("sessID", s.Id),
			logger.Magenta.Field(),
		)
	}
}

// Send 发送proto数据给客户端
func (s *Session) Send(msg proto.Message) bool {
	msgID, err := pb.GetMsgIDS2C(msg)
	if err != nil {
		logger.Warnf("msgIDC2S error:%v", err)
		return false
	}
	return s.SendPB(msgid.MsgIDS2C(msgID), msg)
}

// SendPB 发送proto数据给客户端
func (s *Session) SendPB(msgID msgid.MsgIDS2C, msg proto.Message) bool {
	if msg == nil {
		logger.Warnf("msg is nil")
		return false
	}

	var b []byte
	var err error
	b, err = proto.Marshal(msg)
	if err != nil {
		logger.Warnf("send pb, marshal error:%v", err)
		return false
	}

	s.SendBytes(uint32(msgID), b)
	if IsTraceProto() {
		if msgID != msgid.MsgIDS2C_S2CHeartBeat {
			logger.Infof("%s send [%d]%s data:%v",
				s.String(), msgID, msgid.MsgIDS2C_name[int32(msgID)], msg)
		}
	}
	return true
}

// send
func (s *Session) sendLoop() {
	cache := make([]byte, sendPacketLimit+8)

	for {
		select {
		case p := <-s.out:
			s.sendWithCache(p, cache)
		case <-s.ctrl:
			return
		}
	}
}

func (s *Session) sendWithCache(p *MsgSend, cache []byte) {
	if len(p.Data) > sendPacketLimit {
		cacheTemp := make([]byte, len(p.Data)+8)
		s.rawSend(p, cacheTemp)
	} else {
		s.rawSend(p, cache)
	}
}

func (s *Session) rawSend(p *MsgSend, cache []byte) {
	data := Encode(p.ID, p.Data, s.enCpy, cache)
	err := s.conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		logger.Debugf("send data error, err:%v", err)
	}
}
