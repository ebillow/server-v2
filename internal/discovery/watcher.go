package discovery

import (
	"context"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Watcher 管理服务发现
type Watcher struct {
	client   *clientv3.Client
	prefix   string
	onAdd    func(key, val string)
	onDelete func(key string)
}

// NewWatcher 创建一个服务发现者
func NewWatcher(endpoints []string, prefix string,
	onAdd func(key, val string),
	onDelete func(key string),
) (*Watcher, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &Watcher{client: cli, prefix: prefix, onAdd: onAdd, onDelete: onDelete}, nil
}

//
//
// // GetSvcMetaList 获取某种服务的元数据列表
// func (c *Client) GetSvcMetaList(ctx context.Context, svcName netpack.ServerType) (MetaList, error) {
// 	prefix := BuildServicePrefixPath(svcName)
// 	resp, err := c.EtcdCli.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
// 	if err != nil {
// 		return nil, gerrors.Wrapf(err, "get service meta failed")
// 	}
// 	if len(resp.Kvs) == 0 {
// 		return nil, nil
// 	}
//
// 	var metaList = make([]Meta, 0, len(resp.Kvs))
// 	for _, kv := range resp.Kvs {
// 		var meta Meta
// 		_ = json.Unmarshal(kv.Value, &meta)
// 		metaList = append(metaList, meta)
// 	}
//
// 	return metaList, nil
// }
//
// // GetSvcMeta 获取某个服务的元数据
// func (c *Client) GetSvcMeta(ctx context.Context, svcName netpack.ServerType, svcIdx int) (*Meta, bool, error) {
// 	resp, err := c.EtcdCli.Get(ctx, buildServicePath(svcName, svcIdx))
// 	if err != nil {
// 		return nil, false, gerrors.Wrapf(err, "get service meta failed")
// 	}
// 	if len(resp.Kvs) == 0 {
// 		return nil, false, nil
// 	}
//
// 	kv := resp.Kvs[0]
// 	var meta Meta
// 	_ = json.Unmarshal(kv.Value, &meta)
// 	return &meta, true, nil
// }

// Watch 启动初始拉取和 Watch
func (w *Watcher) Watch() error {
	// 1. 初始化：拉取现有服务
	resp, err := w.client.Get(context.Background(), w.prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		w.onAdd(string(kv.Key), string(kv.Value))
	}

	// 2. Watch 后续变化
	watchCh := w.client.Watch(context.Background(), w.prefix, clientv3.WithPrefix())
	go func() {
		for wr := range watchCh {
			for _, ev := range wr.Events {
				switch ev.Type {
				case mvccpb.PUT:
					w.onAdd(string(ev.Kv.Key), string(ev.Kv.Value))
				case mvccpb.DELETE:
					w.onDelete(string(ev.Kv.Key))
				}
			}
		}
	}()
	return nil
}
