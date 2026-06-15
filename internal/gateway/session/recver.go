package session

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	pb "server/api/pb"
	"server/api/pb/msgid"
	"server/pkg/gerror"
	"server/pkg/gnet/msgq"
	"server/pkg/gnet/trace"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var recvBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, maxRecvMsgSize)
		return &b
	},
}

func (s *Session) readLoop(ctx context.Context, cfg *Config) {
	defer func() {
		s.onLoopExit()
	}()

	if cfg.ReadDeadline > 0 {
		_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
	}

	for {
		mt, r, err := s.conn.NextReader()
		if err != nil {
			s.Close(classifyReadErr(err))
			return
		}
		if cfg.ReadDeadline > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline))
		}

		if mt != websocket.BinaryMessage {
			continue
		}

		bufPtr := recvBufPool.Get().(*[]byte)
		buf := *bufPtr
		var total int

		total, err = s.readFull(r, buf)
		if err != nil {
			recvBufPool.Put(bufPtr)
			s.Close(classifyReadErr(err))
			return
		}

		s.forwardToSrv(buf[:total])
		recvBufPool.Put(bufPtr)

		select {
		case <-ctx.Done():
			s.Close(pb.DisconnectReason_Normal)
			return
		default:
		}
	}
}
func (s *Session) readFull(r io.Reader, buf []byte) (total int, err error) {
	for {
		n, err := r.Read(buf[total:])
		total += n

		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			zap.L().Warn("read payload err", zap.Inline(s), zap.Error(err))
			return 0, err
		}
		if total >= len(buf) {
			zap.L().Warn("message too large", zap.Inline(s), zap.Int("limit", len(buf)))
			return 0, gerror.Newf("message too large:%d > %d", total, len(buf))
		}
	}
}

func classifyReadErr(err error) pb.DisconnectReason {
	if err == nil {
		return pb.DisconnectReason_Normal
	}

	// WebSocket Close 帧：客户端主动正常关闭 ----
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		switch closeErr.Code {
		case websocket.CloseNormalClosure, // 1000 正常关闭
			websocket.CloseGoingAway,        // 1001 页面离开/切后台
			websocket.CloseNoStatusReceived, // 1005 无状态码（合成）
			websocket.CloseAbnormalClosure:  // 1006 异常关闭（合成）
			zap.L().Info("client closed normally",
				zap.Int("code", closeErr.Code),
				zap.String("text", closeErr.Text),
			)
			return pb.DisconnectReason_Normal
		default:
			// 客户端异常关闭 (如 1002 ProtocolError, 1011 InternalErr 等)
			zap.L().Warn("client closed with error code",
				zap.Int("code", closeErr.Code),
				zap.String("text", closeErr.Text),
			)
			return pb.DisconnectReason_NetErr
		}
	}

	// 超时：ReadDeadline 到期 ----
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		zap.L().Warn("read timeout", zap.Error(err))
		return pb.DisconnectReason_Heartbeat
	}

	// 连接被对端重置 / 管道破裂 ----
	// Linux: "connection reset by peer"  /  "broken pipe"
	// Windows: wsarecv 相关错误
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr *os.SyscallError
		if errors.As(opErr.Err, &sysErr) {
			errMsg := strings.ToLower(sysErr.Error())
			if strings.Contains(errMsg, "connection reset") ||
				strings.Contains(errMsg, "broken pipe") ||
				strings.Contains(errMsg, "wsarecv") {
				zap.L().Info("connection reset by peer", zap.Error(err))
				return pb.DisconnectReason_Normal // 客户端强杀进程，视为正常离开
			}
		}
	}

	// ----  消息体过大（readFull 返回的错误）----
	if strings.Contains(err.Error(), "message too large") {
		zap.L().Warn("message too large", zap.Error(err))
		return pb.DisconnectReason_Limit
	}

	// 兜底：未知读错误 ----
	zap.L().Warn("unknown read error", zap.Error(err))
	return pb.DisconnectReason_Unknown
}

func decode(src []byte) (msgID uint32, data []byte, err error) {
	const minDataLen = 4
	if len(src) < minDataLen {
		return 0, nil, errors.New("packet head < 4")
	}

	msgID = binary.BigEndian.Uint32(src[0:4])
	data = src[4:]
	return
}

func (s *Session) forwardToSrv(src []byte) {
	msgID, data, err := decode(src)
	if err != nil {
		zap.L().Warn("read packet err", zap.Inline(s), zap.Error(err))
		s.Close(pb.DisconnectReason_DecodeErr)
		return
	}

	if msgID <= uint32(msgid.MsgIDMax_S2SMax) {
		zap.L().Debug("invalid msg id", zap.Uint32("msgID", msgID), zap.Inline(s))
		return
	}

	serType := pb.Server(msgID / 100000)
	serID := s.getSerID(serType)
	err = msgq.Q.SendToNode(serType, serID, msgID, data, 0, s.Id)
	if err != nil {
		zap.L().Warn("send to server err"+msgid.MsgIDC2S_name[int32(msgID)],
			zap.Uint32("msgID", msgID),
			zap.String("to", pb.Server_name[int32(serType)]),
			zap.Uint8("idx", serID),
			zap.Inline(s),
		)
		return
	}
	if trace.Rule.ShouldLog(msgID, 0, s.Id) {
		zap.L().Info(">>> to server: "+msgid.MsgIDC2S_name[int32(msgID)],
			zap.Uint32("msgID", msgID),
			zap.String("to", pb.Server_name[int32(serType)]),
			zap.Uint8("idx", serID),
			zap.Inline(s),
		)
	}
}
