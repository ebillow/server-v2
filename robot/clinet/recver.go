package clinet

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/pkg/pb/msgid"
)

func RegistryMsg(msgID msgid.MsgIDS2C, cf func() proto.Message, df func(msg proto.Message, s *Session)) {
	cliMsgHandler.Register(uint32(msgID), cf, df)
}

// recv
func (s *Session) recvLoop(cfg *Config) {
	zap.S().Debug("recv loop start")
	for {
		// if cfg.ReadDeadline > 0 {
		//	_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
		// }
		mt, data, err := s.conn.ReadMessage()
		if err != nil {
			zap.S().Debugf("%d read message err:%v", s.Id, err)
			return
		}
		if mt == websocket.CloseMessage {
			zap.S().Debugf("%s connection close: recv close message", s.String())
			return
		} else if mt != websocket.BinaryMessage {
			continue
		}

		select {
		case s.in <- data:
		case <-s.ctrl:
			zap.S().Debugf("%s connection close by component", s.String())
			return
		}
	}
}

func (s *Session) onRecvCliMsg(data []byte) {
	p, err := newReader(data, s.DeCyp)
	if err != nil {
		zap.S().Warnf("%s read packet err:%v", s.String(), err)
		s.Close()
		return
	}

	msgID := p.GetMsgID()
	// seqNum := p.GetSeqNum() //only c2s

	if msgID != uint32(msgid.MsgIDC2S_C2SInit) && !s.flag.Has(SesInit) {
		zap.S().Errorf("%s not init", s.String())
		s.Close()
		return
	}

	if !cliMsgHandler.Handle(msgID, p.GetData(), s) {
	}
}

var cliMsgHandler = NewRoute(65536)
