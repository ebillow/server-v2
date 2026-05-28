package msgq

import (
	"errors"
	"fmt"
	"server/api/pb"
	"server/pkg/flag"
	"server/pkg/gerror"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

var Q DataBus

const (
	SvcTypeMax = 64
	SvcIDMax   = 64
)

const (
	headerSize = 4 + 8 + 8 + 1 + 1 + 1 + 1 + 1
	batchCount = 500
	buffSize   = 1024 * batchCount
)

var (
	ErrClosed = gerror.New("msgq closed")
	ErrArg    = gerror.New("msgq invalid argument")
)

type DataBus struct {
	conn    *nats.Conn
	rpcConn *nats.Conn
	serType uint8
	serID   uint8

	closed atomic.Bool

	pubIDXs   [SvcTypeMax][SvcIDMax]atomic.Pointer[PubBatcher]
	pubIDXMtx sync.Mutex

	pubGroup    [SvcTypeMax]atomic.Pointer[PubBatcher]
	pubGroupMtx sync.Mutex

	pubAll    [SvcTypeMax]atomic.Pointer[PubBatcher]
	pubAllMtx sync.Mutex
}

func (bs *DataBus) Init(connStr string, serType pb.Server, serID uint8, options ...nats.Option) error {
	conn, err := setupNatsConn(connStr, serType, serID, options...)
	if err != nil {
		return err
	}
	bs.conn = conn
	conn, err = setupNatsConn(connStr, serType, serID, options...)
	if err != nil {
		return err
	}
	bs.rpcConn = conn
	bs.serType = uint8(serType)
	bs.serID = serID
	return nil
}

func setupNatsConn(connectString string, svcType pb.Server, svcID uint8, options ...nats.Option) (*nats.Conn, error) {
	natsOptions := append(
		options,
		nats.Name(fmt.Sprintf("%s-%d", flag.SrvName(svcType), svcID)),
		nats.PingInterval(time.Second*12), nats.MaxPingsOutstanding(3),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.Timeout(3*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			SetNatsConnStatus(nc.ConnectedUrl(), 0)
			zap.S().Errorf("disconnected from nats! Reason: %q\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			SetNatsConnStatus(nc.ConnectedUrl(), 1)
			IncNatsReconnect(nc.ConnectedUrl())
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
				IncSlowConsumer(sub.Subject)
				dropped, _ := sub.Dropped()
				zap.S().Warnf("nats slow consumer on subject %q: dropped %d messages\n",
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
	SetNatsConnStatus(connectString, 1)
	return nc, nil
}

var rpcSubCache sync.Map

func rpcIdxSubjectName(serType pb.Server, serID uint8) string {
	key := (uint64(serType) << 32) | uint64(serID)
	if val, ok := rpcSubCache.Load(key); ok {
		return val.(string)
	}

	str := "rpc." + flag.SrvName(serType) + ".idx." + strconv.Itoa(int(serID))
	rpcSubCache.Store(key, str)
	return str
}

func idxSubjectName(serType pb.Server, serID uint8) string {
	return "msg." + flag.SrvName(serType) + ".idx." + strconv.Itoa(int(serID))
}

func groupSubjectName(serType pb.Server) string {
	return "msg." + flag.SrvName(serType) + ".group"
}

func allSubjectName(serType pb.Server) string {
	return "msg." + flag.SrvName(serType) + ".all"
}
