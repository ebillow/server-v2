package msgq

import (
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"server/pkg/gnet/trace"
	"server/pkg/pb"
	"time"
)

var Q DataBus

type DataBus struct {
	conn    *nats.Conn
	serType pb.Server
	serID   int32
	Tracer  *trace.TraceRule
}

func (bs *DataBus) Init(connStr string, serType pb.Server, serID int32, options ...nats.Option) error {
	conn, err := setupNatsConn(connStr, options...)
	if err != nil {
		return err
	}
	bs.conn = conn
	bs.serType = serType
	bs.serID = serID
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
