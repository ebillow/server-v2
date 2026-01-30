package msgq

import (
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"server/pkg/gnet/trace"
	"strconv"
	"time"
)

var Q DataBus

type DataBus struct {
	conn    *nats.Conn
	serName string
	serID   string
	Tracer  *trace.TraceRule
}

func (bs *DataBus) Init(connStr string, serName string, serID int32, options ...nats.Option) error {
	conn, err := setupNatsConn(connStr, options...)
	if err != nil {
		return err
	}
	bs.conn = conn
	bs.serName = serName
	bs.serID = strconv.Itoa(int(serID))
	return nil
}

func setupNatsConn(connectString string, options ...nats.Option) (*nats.Conn, error) {
	natsOptions := append(
		options,
		nats.PingInterval(time.Second*12), nats.MaxPingsOutstanding(3),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			zap.S().Errorf("disconnected from nats! Reason: %q\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			zap.S().Infof("reconnected to nats server %s with address %s in cluster %s!", nc.ConnectedServerName(), nc.ConnectedAddr(), nc.ConnectedClusterName())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			err := nc.LastError()
			if err == nil {
				zap.S().Warn("nats connection closed with no error.")
				return
			}

			zap.S().Errorf("nats connection closed. reason: %q", nc.LastError())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			if errors.Is(err, nats.ErrSlowConsumer) {
				dropped, _ := sub.Dropped()
				zap.S().Warn("nats slow consumer on subject %q: dropped %d messages\n",
					sub.Subject, dropped)
			} else {
				zap.S().Errorf(err.Error())
			}
		}),
	)

	nc, err := nats.Connect(connectString, natsOptions...)
	if err != nil {
		return nil, err
	}
	return nc, nil
}

func getIndexSubject(serName string, serID int32) string {
	return fmt.Sprintf("msg.%s.idx.%d", serName, serID)
}

func getGroupSubject(serName string) string {
	return fmt.Sprintf("msg.%s.group", serName)
}

func getAllSubject(serName string) string {
	return fmt.Sprintf("msg.%s.all", serName)
}

const (
	FServerName = "srv_name"
	FServerID   = "srv_id"
	FMsgID      = "msg_id"
	FRoleID     = "role_id"
	FSessionID  = "ses_id"
)

func SessionID(msg *nats.Msg) uint64 {
	i, _ := strconv.Atoi(msg.Header.Get(FSessionID))
	return uint64(i)
}

func RoleID(msg *nats.Msg) uint64 {
	i, _ := strconv.Atoi(msg.Header.Get(FRoleID))
	return uint64(i)
}

func MsgID(msg *nats.Msg) uint32 {
	i, _ := strconv.Atoi(msg.Header.Get(FMsgID))
	return uint32(i)
}

func ServerName(msg *nats.Msg) string {
	return msg.Header.Get(FServerName)
}

func ServerID(msg *nats.Msg) int32 {
	i, _ := strconv.Atoi(msg.Header.Get(FServerID))
	return int32(i)
}
