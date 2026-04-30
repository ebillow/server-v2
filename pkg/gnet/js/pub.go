package js

import (
	"server/pkg/pb"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func (jt *JetStream) Send(serType pb.Server, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	natsMsg := &pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerID:   jt.serID,
		SerType: jt.serType,
		RoleID:  roleID,
		SesID:   sesID,
	}
	b, err := proto.Marshal(natsMsg)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: getIndexSubject(serType, serID),
		Data:    b,
	}
	// 设置 MsgId 也就是 Nats-Msg-Id Header，JS 会根据这个ID在配置窗口期内去重
	// msg.Header.Set(jetstream.MsgIDHeader, ev.EventID)

	// 异步发布，不阻塞主线程
	_, err = jt.JS.PublishMsgAsync(msg)
	if err != nil {
		return err
	}

	// jt.ack <- ackF
	return err
}

func (jt *JetStream) SendAny(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	natsMsg := &pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerID:   jt.serID,
		SerType: jt.serType,
		RoleID:  roleID,
		SesID:   sesID,
	}
	b, err := proto.Marshal(natsMsg)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: getGroupSubject(serType),
		Data:    b,
	}
	// 异步发布，不阻塞主线程
	ackF, err := jt.JS.PublishMsgAsync(msg)
	if err != nil {
		return err
	}

	jt.ack <- ackF
	return err
}

func (jt *JetStream) checkAck() {
	jt.ack = make(chan jetstream.PubAckFuture, 20480)
	go func() {
		for v := range jt.ack {
			select {
			case <-v.Ok():
			case err := <-v.Err():
				zap.L().Error("jt pub error", zap.Error(err))
			case <-time.After(time.Second * 5):
				zap.L().Error("jt pub timeout")
			}
		}
	}()
}
