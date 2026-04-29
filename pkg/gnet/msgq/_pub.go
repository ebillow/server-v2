package msgq

import (
	"server/pkg/gerror"
	"server/pkg/gnet/codec"
	"server/pkg/gnet/gctx"
	"server/pkg/pb"
)

// Send 指定发送
func (bs *DataBus) Send(serType pb.Server, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, bp, err := codec.Encode(gctx.Context{
		MsgID:   msgID,
		Data:    data,
		SerID:   bs.serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: 0,
	})
	if err != nil {
		return gerror.Wrap(err, "encode err:")
	}
	err = bs.conn.Publish(getIndexSubject(serType, serID), out)
	if err != nil {
		codec.FreeBuffer(bp)
		return gerror.Wrap(err, "publish err:")
	}
	codec.FreeBuffer(bp)
	return nil
}

func (bs *DataBus) ForwardToRole(serType pb.Server, serID int32, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, bp, err := codec.Encode(gctx.Context{
		MsgID:   msgID,
		Data:    data,
		SerID:   bs.serID,
		SerType: bs.serType,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: 1,
	})
	if err != nil {
		return gerror.Wrap(err, "encode err:")
	}
	err = bs.conn.Publish(getIndexSubject(serType, serID), out)
	if err != nil {
		codec.FreeBuffer(bp)
		return gerror.Wrap(err, "publish err:")
	}
	codec.FreeBuffer(bp)
	return nil
}

// SendAny 组发送. 随机一个能收到
func (bs *DataBus) SendAny(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, bp, err := codec.Encode(gctx.Context{
		MsgID:   msgID,
		Data:    data,
		SerType: bs.serType,
		SerID:   bs.serID,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: 0,
	})
	if err != nil {
		return gerror.Wrap(err, "encode err:")
	}
	err = bs.conn.Publish(getGroupSubject(serType), out)
	if err != nil {
		codec.FreeBuffer(bp)
		return gerror.Wrap(err, "publish err:")
	}
	codec.FreeBuffer(bp)
	return nil
}

// SendAll 所有的 serName 服节点都能收到
func (bs *DataBus) SendAll(serType pb.Server, msgID uint32, data []byte, roleID uint64, sesID uint64) error {
	out, bp, err := codec.Encode(gctx.Context{
		MsgID:   msgID,
		Data:    data,
		SerType: bs.serType,
		SerID:   bs.serID,
		RoleID:  roleID,
		SesID:   sesID,
		Forward: 0,
	})
	if err != nil {
		return gerror.Wrap(err, "encode err:")
	}
	err = bs.conn.Publish(getAllSubject(serType), out)
	if err != nil {
		codec.FreeBuffer(bp)
		return gerror.Wrap(err, "publish err:")
	}
	codec.FreeBuffer(bp)
	return nil
}
