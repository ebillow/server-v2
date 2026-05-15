package discovery

import (
	"math/rand/v2"
	"sync"
)

type NodeMeta struct {
	NodeID int32
	Load   int32
}

type NodeGroup struct {
	mtx     sync.RWMutex
	nodes   map[int32]NodeMeta
	nodeIDs []int32
}

func NewNodeGroup() *NodeGroup {
	return &NodeGroup{
		nodes: make(map[int32]NodeMeta),
	}
}

// Node 缓存结构重建
func (o *NodeGroup) rebuildNodeCache() {
	o.nodeIDs = make([]int32, 0, len(o.nodes))
	for id := range o.nodes {
		o.nodeIDs = append(o.nodeIDs, id)
	}
}

func (o *NodeGroup) Add(meta NodeMeta) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	o.nodes[meta.NodeID] = meta
	o.rebuildNodeCache()
}

func (o *NodeGroup) RangeReadOnly(f func(m NodeMeta)) {
	o.mtx.RLock()
	defer o.mtx.RUnlock()

	for _, v := range o.nodes {
		f(v)
	}
}

func (o *NodeGroup) GetLoad(id int32) (int32, bool) {
	o.mtx.RLock()
	defer o.mtx.RUnlock() // 必须使用 defer

	meta, ok := o.nodes[id]
	return meta.Load, ok
}

func (o *NodeGroup) Delete(serID int32) bool {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	delete(o.nodes, serID)
	o.rebuildNodeCache()
	return len(o.nodes) == 0
}

// UpdateLoad 更新特定节点的负载
func (o *NodeGroup) UpdateLoad(serID int32, load int32) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	if meta, ok := o.nodes[serID]; ok {
		meta.Load = load
		o.nodes[serID] = meta
	}
}

// SelectNode 基于本地缓存获取负载最小的节点
func (o *NodeGroup) SelectNode() (int32, bool) {
	o.mtx.RLock()
	defer o.mtx.RUnlock()

	n := len(o.nodeIDs)
	if n == 0 {
		return 0, false
	}
	if n == 1 {
		return o.nodeIDs[0], true
	}

	// 随机选两个不同的节点
	i := rand.IntN(n)
	j := rand.IntN(n - 1)
	if j >= i {
		j++
	}

	node1 := o.nodes[o.nodeIDs[i]]
	node2 := o.nodes[o.nodeIDs[j]]

	// 返回负载较小的那个
	if node1.Load <= node2.Load {
		return node1.NodeID, true
	}
	return node2.NodeID, true
}

func (o *NodeGroup) Exists(id int32) bool {
	o.mtx.RLock()
	o.mtx.RUnlock()
	_, ok := o.nodes[id]
	return ok
}
