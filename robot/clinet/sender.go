package clinet

import (
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"server/pkg/logger"
	"server/pkg/pb/msgid"
	"server/pkg/thread"
)

// send 发送数据给客户端,非线程安全
func (s *Session) Send(msgID msgid.MsgIDC2S, data []byte) bool {
	select {
	case s.out <- newPkgWriter(uint32(msgID), data):
		return true
	default:
		logger.Warnf("%s send queue full", s.String())
		return false
	}
}

// SendPB 发送proto数据给客户端,非线程安全
func (s *Session) SendPB(msgID msgid.MsgIDC2S, msgData proto.Message) bool {
	var b []byte
	if msgData != nil {
		var err error
		b, err = proto.Marshal(msgData)
		if err != nil {
			logger.Warnf("send pb, marshal error:%v", err)
			return false
		}
	}

	if s.Send(msgID, b) {
		if msgID != msgid.MsgIDC2S_C2SHeartBeat {
			logger.Tracef("%s send [%d]%s data:%s",
				s.String(), msgID, msgid.MsgIDC2S_name[int32(msgID)], msgData)
		}
		return true
	} else {
		return false
	}
}

// send
func (s *Session) sendLoop(cfg *Config) {
	defer func() {
		if err := recover(); err != nil {
			thread.PrintStack(err)
		}
		logger.Debug("send loop stop")
	}()

	cache := make([]byte, sendPacketLimit+10) // 内存占用点

	for {
		select {
		case p := <-s.out:
			s.sendWithCache(p, cache)
		case <-s.ctrl:
			// for p := range s.out{//关闭时，有些时候需要把消息发给客户端
			//	s.sendWithCache(p, active_role)
			// }
			return
		}
	}
}

func (s *Session) sendWithCache(p *pkgWriter, cache []byte) {
	if len(p.data) > sendPacketLimit {
		logger.Tracef("send %d data size:%d > limit", p.msgId, len(p.data))
		cacheTemp := make([]byte, len(p.data)+10)
		s.rawSend(p, cacheTemp)
	} else {
		s.rawSend(p, cache)
	}
}

func (s *Session) rawSend(p *pkgWriter, cache []byte) {
	s.pkgSend++
	len := p.Write(cache, s.pkgSend, s.EnCpy)
	err := s.conn.WriteMessage(websocket.BinaryMessage, cache[:len])
	if err != nil {
		logger.Warnf("send data error, err:%v", err)
	}
}
