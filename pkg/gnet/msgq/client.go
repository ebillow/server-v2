package msgq

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"server/pkg/util"
)

// Send 指定发送
func (bs *DataBus) Send(serName string, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) {
	msg := nats.NewMsg(getIndexSubject(serName, serID))
	bs.setHeader(msg, msgID, roleID, sesID)
	msg.Data = data

	err := bs.conn.PublishMsg(msg)
	if err != nil {
		zap.L().Warn("[msg] SendIndex error", zap.Error(err), zap.String("serName", serName), zap.Int32("serID", serID))
	}
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serName string, msgID uint32, data []byte, roleID uint64, sesID uint64) {
	msg := nats.NewMsg(getGroupSubject(serName))
	bs.setHeader(msg, msgID, roleID, sesID)
	msg.Data = data

	err := bs.conn.PublishMsg(msg)
	if err != nil {
		zap.L().Warn("[msg] SendAny error", zap.Error(err), zap.String("serName", serName))
	}
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serName string, msgID uint32, data []byte, roleID uint64, sesID uint64) {
	msg := nats.NewMsg(getAllSubject(serName))
	bs.setHeader(msg, msgID, roleID, sesID)
	msg.Data = data

	err := bs.conn.PublishMsg(msg)
	if err != nil {
		zap.L().Warn("[msg] SendAll error", zap.Error(err), zap.String("serName", serName))
	}
}

func (bs *DataBus) setHeader(msg *nats.Msg, msgID uint32, roleID uint64, sesID uint64) {
	msg.Header.Set(FServerName, bs.serName)
	msg.Header.Set(FServerID, bs.serID)
	msg.Header.Set(FMsgID, util.ToString(msgID))
	if roleID != 0 {
		msg.Header.Set(FRoleID, util.ToString(roleID))
	}
	if sesID != 0 {
		msg.Header.Set(FSessionID, util.ToString(sesID))
	}
}
