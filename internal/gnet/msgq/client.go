package msgq

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"server/internal/util"
)

// Send 指定发送
func (bs *DataBus) Send(serName string, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) {
	msg := nats.NewMsg(getIndexSubject(serName, serID))
	msg.Header.Set("ser_name", bs.serName)
	msg.Header.Set("ser_id", bs.serID)
	msg.Header.Set("msg_id", util.ToString(msgID))
	if roleID != 0 {
		msg.Header.Set("role", util.ToString(roleID))
	}
	if sesID != 0 {
		msg.Header.Set("session", util.ToString(sesID))
	}
	msg.Data = data

	err := bs.conn.PublishMsg(msg)
	if err != nil {
		zap.L().Warn("[msg] SendIndex error", zap.Error(err), zap.String("serName", serName), zap.Int32("serID", serID))
	}
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serName string, msgID int32, data []byte, roleID uint64, sesID uint64) {
	msg := nats.NewMsg(getGroupSubject(serName))
	msg.Header.Set("ser_name", bs.serName)
	msg.Header.Set("ser_id", bs.serID)
	msg.Header.Set("msg_id", util.ToString(msgID))
	if roleID != 0 {
		msg.Header.Set("role", util.ToString(roleID))
	}
	if sesID != 0 {
		msg.Header.Set("session", util.ToString(sesID))
	}

	msg.Data = data

	err := bs.conn.PublishMsg(msg)
	if err != nil {
		zap.L().Warn("[msg] SendAny error", zap.Error(err), zap.String("serName", serName))
	}
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serName string, msgID int32, data []byte, roleID uint64, sesID uint64) {
	msg := nats.NewMsg(getAllSubject(serName))
	msg.Header.Set("ser_name", bs.serName)
	msg.Header.Set("ser_id", bs.serID)
	msg.Header.Set("msg_id", util.ToString(msgID))
	if roleID != 0 {
		msg.Header.Set("role", util.ToString(roleID))
	}
	if sesID != 0 {
		msg.Header.Set("session", util.ToString(sesID))
	}

	msg.Data = data

	err := bs.conn.PublishMsg(msg)
	if err != nil {
		zap.L().Warn("[msg] SendAll error", zap.Error(err), zap.String("serName", serName))
	}
}
