package logic

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server/gateway/session"
	"server/pkg/pb/msgid"
)

// SendToCli	发送数据给客户端,用于协程外发消息
func SendToCli(cliSesId uint64, msgID msgid.MsgIDS2C, msg proto.Message) {
	ses := session.GetCliSession(cliSesId)
	if ses != nil {
		ses.SendPB(msgID, msg)
	}
}

func SendToAll(msgID msgid.MsgIDS2C, msg proto.Message) {
	b, err := proto.Marshal(msg)
	if err != nil {
		zap.S().Warnf("send pb, marshal error:%v", err)
		return
	}

	session.CliSess.Range(func(k, v interface{}) bool {
		ses := v.(*session.Session)
		ses.SendBytes(uint32(msgID), b)
		return true
	})
}

func SendToSomeOne(msgID msgid.MsgIDS2C, msg proto.Message, cliSess map[uint64]bool) {
	b, err := proto.Marshal(msg)
	if err != nil {
		zap.S().Warnf("send pb, marshal error:%v", err)
		return
	}
	for k := range cliSess {
		ses := session.GetCliSession(k)
		if ses != nil {
			ses.SendBytes(uint32(msgID), b)
		}
	}
}
