package discovery

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"
)

type nodeMeta struct {
	NodeID int32
	Load   atomic.Int32
}

type nodeGroup struct {
	mu    sync.Mutex
	state atomic.Pointer[nodeGroupState]
}

type nodeGroupState struct {
	nodes   map[int32]*nodeMeta
	nodeIDs []int32
}

func newNodeGroup() *nodeGroup {
	ng := &nodeGroup{}
	ng.state.Store(&nodeGroupState{
		nodes:   make(map[int32]*nodeMeta),
		nodeIDs: make([]int32, 0),
	})
	return ng
}

// Add meta要初始化load
func (o *nodeGroup) Add(msg Node) {
	o.mu.Lock()
	defer o.mu.Unlock()

	oldState := o.state.Load()

	newNodes := make(map[int32]*nodeMeta, len(oldState.nodes)+1)
	for k, v := range oldState.nodes {
		newNodes[k] = v
	}

	meta := &nodeMeta{
		NodeID: msg.NodeID,
	}
	meta.Load.Store(msg.Load)

	newNodes[meta.NodeID] = meta

	newNodesIDs := make([]int32, 0, len(newNodes))
	for id := range newNodes {
		newNodesIDs = append(newNodesIDs, id)
	}

	o.state.Store(&nodeGroupState{
		nodes:   newNodes,
		nodeIDs: newNodesIDs,
	})
}

func (o *nodeGroup) Delete(nodeID int32) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	oldState := o.state.Load()
	if _, exits := oldState.nodes[nodeID]; !exits {
		return len(oldState.nodes) == 0
	}

	newNodes := make(map[int32]*nodeMeta, len(oldState.nodes)-1)
	for k, v := range oldState.nodes {
		if k != nodeID {
			newNodes[k] = v
		}
	}

	newNodesIDs := make([]int32, 0, len(newNodes))
	for id := range newNodes {
		newNodesIDs = append(newNodesIDs, id)
	}
	o.state.Store(&nodeGroupState{
		nodes:   newNodes,
		nodeIDs: newNodesIDs,
	})
	return len(newNodes) == 0
}

func (o *nodeGroup) RangeReadOnly(f func(nodeID, load int32)) {
	state := o.state.Load()

	for id, v := range state.nodes {
		f(id, v.Load.Load())
	}
}

func (o *nodeGroup) GetLoad(id int32) (int32, bool) {
	state := o.state.Load()
	if meta, ok := state.nodes[id]; ok {
		return meta.Load.Load(), true
	}
	return 0, false
}

// UpdateLoad 更新特定节点的负载
func (o *nodeGroup) UpdateLoad(nodeID int32, load int32) {
	state := o.state.Load()
	if meta, ok := state.nodes[nodeID]; ok {
		meta.Load.Store(load)
	}
}

// SelectNode 基于本地缓存获取负载最小的节点
func (o *nodeGroup) SelectNode() (int32, bool) {
	state := o.state.Load()

	n := len(state.nodeIDs)
	if n == 0 {
		return 0, false
	}
	if n == 1 {
		return state.nodeIDs[0], true
	}

	// 随机选两个不同的节点
	i := rand.IntN(n)
	j := rand.IntN(n - 1)
	if j >= i {
		j++
	}

	node1 := state.nodes[state.nodeIDs[i]]
	node2 := state.nodes[state.nodeIDs[j]]

	// 返回负载较小的那个
	if node1.Load.Load() <= node2.Load.Load() {
		return node1.NodeID, true
	}
	return node2.NodeID, true
}

func (o *nodeGroup) Exists(id int32) bool {
	state := o.state.Load()
	_, ok := state.nodes[id]
	return ok
}

func (o *nodeGroup) AllNodeIDs() []int32 {
	state := o.state.Load()
	ids := make([]int32, len(state.nodeIDs))
	copy(ids, state.nodeIDs)
	return ids
}
