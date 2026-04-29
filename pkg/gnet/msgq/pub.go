package msgq

import (
	"server/pkg/gnet/gctx"
	"server/pkg/pb"
)

func (bs *DataBus) getIdxPubBatcher(serType pb.Server, serID int32) *PubBatcher {
	key := (uint64(serType) << 32) | uint64(uint32(serID))
	val, ok := bs.pubBatchers.Load(key)
	if !ok {
		subject := getIndexSubject(serType, serID)
		tb := NewPubBatcher(subject, bs.conn)
		actual, _ := bs.pubBatchers.LoadOrStore(key, tb)
		val = actual
	}
	return val.(*PubBatcher)
}

func (bs *DataBus) getGroupPubBatcher(serType pb.Server) *PubBatcher {
	key := uint64(serType)
	val, ok := bs.pubBatchers.Load(key)
	if !ok {
		subject := getGroupSubject(serType)
		tb := NewPubBatcher(subject, bs.conn)
		actual, _ := bs.pubBatchers.LoadOrStore(key, tb)
		val = actual
	}
	return val.(*PubBatcher)
}

func (bs *DataBus) getAllPubBatcher(serType pb.Server) *PubBatcher {
	key := uint64(serType)
	val, ok := bs.pubBatchers.Load(key)
	if !ok {
		subject := getAllSubject(serType)
		tb := NewPubBatcher(subject, bs.conn)
		actual, _ := bs.pubBatchers.LoadOrStore(key, tb)
		val = actual
	}
	return val.(*PubBatcher)
}

// Send 指定发送
func (bs *DataBus) Send(serType pb.Server, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	bs.getIdxPubBatcher(serType, serID).Add(gctx.Context{
		MsgID:   msgID,
		Data:    data,
		SerID:   bs.serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: 0,
	})
	return nil
}

func (bs *DataBus) ForwardToRole(serType pb.Server, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	bs.getIdxPubBatcher(serType, serID).Add(gctx.Context{
		MsgID:   msgID,
		Data:    data,
		SerID:   bs.serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: 1,
	})

	return nil
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	bs.getGroupPubBatcher(serType).Add(gctx.Context{
		MsgID:   msgID,
		Data:    data,
		SerType: bs.serType,
		SerID:   bs.serID,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: 0,
	})
	return nil
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	bs.getAllPubBatcher(serType).Add(gctx.Context{
		MsgID:   msgID,
		Data:    data,
		SerType: bs.serType,
		SerID:   bs.serID,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: 0,
	})

	return nil
}
