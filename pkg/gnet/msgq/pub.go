package msgq

import (
	"github.com/pkg/errors"
	"server/pkg/gnet/codec"
	"server/pkg/pb"
)

// Send 指定发送
func (bs *DataBus) Send(serType pb.Server, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, bp, err := codec.Encode(&pb.NatsMsg{ // 可以考虑不用proto，直接写[]byte
		MsgID:   msgID,
		Data:    data,
		SerID:   bs.serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: false,
	})
	if err != nil {
		return errors.Wrap(err, "encode err:")
	}
	err = bs.conn.Publish(getIndexSubject(serType, serID), out)
	if err != nil {
		codec.FreeBuffer(bp)
		return errors.Wrap(err, "publish err:")
	}
	codec.FreeBuffer(bp)
	return nil
}

func (bs *DataBus) ForwardToRole(serType pb.Server, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, bp, err := codec.Encode(&pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerID:   bs.serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: true,
	})
	if err != nil {
		return errors.Wrap(err, "encode err:")
	}
	err = bs.conn.Publish(getIndexSubject(serType, serID), out)
	if err != nil {
		codec.FreeBuffer(bp)
		return errors.Wrap(err, "publish err:")
	}
	codec.FreeBuffer(bp)
	return nil
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, bp, err := codec.Encode(&pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerType: bs.serType,
		SerID:   bs.serID,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: false,
	})
	if err != nil {
		return errors.Wrap(err, "encode err:")
	}
	err = bs.conn.Publish(getGroupSubject(serType), out)
	if err != nil {
		codec.FreeBuffer(bp)
		return errors.Wrap(err, "publish err:")
	}
	codec.FreeBuffer(bp)
	return nil
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, bp, err := codec.Encode(&pb.NatsMsg{
		MsgID:   msgID,
		Data:    data,
		SerType: bs.serType,
		SerID:   bs.serID,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: false,
	})
	if err != nil {
		return errors.Wrap(err, "encode err:")
	}
	err = bs.conn.Publish(getAllSubject(serType), out)
	if err != nil {
		codec.FreeBuffer(bp)
		return errors.Wrap(err, "publish err:")
	}
	codec.FreeBuffer(bp)
	return nil
}
