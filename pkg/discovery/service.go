package discovery

import (
	"encoding/json"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"server/pkg/flag"
	"server/pkg/pb"
	"time"
)

const Prefix = "/services/"

type Meta struct {
	SerID int32
}

var (
	register  *SDRegister
	discovery *Discovery
	cli       *clientv3.Client
)

func Init(endpoints []string) error {
	var err error
	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		zap.L().Error("create etcd service failed", zap.Error(err))
		return err
	}

	return nil
}

func Register(serType pb.Server, serID int32) error {
	b, err := json.Marshal(&Meta{
		SerID: serID,
	})
	if err != nil {
		return err
	}
	register = newRegister(cli, fmt.Sprintf("%s%s_%d", Prefix, flag.SrvName(serType), serID), string(b), 5)
	return register.register()
}

func Watch() {
	discovery = newDiscovery(cli, Prefix)
}

func Close() {
	if register != nil {
		register.close()
	}
}

func Exist(serName string, id int32) bool {
	return discovery.exist(serName, id)
}

func Pick(serName string) (id int32, ok bool) {
	return discovery.pick(serName)
}
