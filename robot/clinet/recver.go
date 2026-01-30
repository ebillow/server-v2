package clinet

import (
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"server/pkg/logger"

	"server/pkg/pb/msgid"
)

func RegistryMsg(msgID msgid.MsgIDS2C, cf func() proto.Message, df func(msg proto.Message, s *Session)) {
	cliMsgHandler.Register(uint32(msgID), cf, df)
}

// recv
func (s *Session) recvLoop(cfg *Config) {
	logger.Debug("recv loop start")
	for {
		// if cfg.ReadDeadline > 0 {
		//	_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
		// }
		mt, data, err := s.conn.ReadMessage()
		if err != nil {
			logger.Debugf("%d read message err:%v", s.Id, err)
			return
		}
		if mt == websocket.CloseMessage {
			logger.Debugf("%s connection close: recv close message", s.String())
			return
		} else if mt != websocket.BinaryMessage {
			continue
		}

		select {
		case s.in <- data:
		case <-s.ctrl:
			logger.Debugf("%s connection close by component", s.String())
			return
		}
	}
}

func (s *Session) onRecvCliMsg(data []byte) {
	p, err := newReader(data, s.DeCyp)
	if err != nil {
		logger.Warnf("%s read packet err:%v", s.String(), err)
		s.Close()
		return
	}

	msgID := p.GetMsgID()
	// seqNum := p.GetSeqNum() //only c2s

	if msgID != uint32(msgid.MsgIDC2S_C2SInit) && !s.flag.Has(SesInit) {
		logger.Errorf("%s not init", s.String())
		s.Close()
		return
	}

	if !cliMsgHandler.Handle(msgID, p.GetData(), s) {
	}
}

var cliMsgHandler = NewRoute(65536)
