package ui

import (
	"context"
	"sync"
	"time"

	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
)

type ViewModelInterface interface {
	BuildTreeData() *TreeNode
	HasTreeChanged(*TreeNode) bool
	FetchTopology() error
	Updates() <-chan struct{}
	StartAutoRefresh(ctx context.Context, interval time.Duration)
}

type TreeNode struct {
	Text     string
	Children []*TreeNode
}

type ViewModel struct {
	Client     rabbitmq.ClientInterface
	mu         sync.RWMutex
	topology   *rabbitmq.Topology
	updateChan chan struct{}
	lastTree   *TreeNode
}

func NewViewModel(client rabbitmq.ClientInterface) *ViewModel {
	return &ViewModel{
		Client:     client,
		updateChan: make(chan struct{}, 1),
	}
}

func (vm *ViewModel) FetchTopology() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	topology, err := vm.Client.FetchTopology()
	if err != nil {
		return err
	}
	vm.topology = topology
	vm.signalUpdate()
	return nil
}

func (vm *ViewModel) StartAutoRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = vm.FetchTopology()
		case <-ctx.Done():
			return
		}
	}
}

func (vm *ViewModel) GetTopology() *rabbitmq.Topology {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.topology
}

func (vm *ViewModel) Updates() <-chan struct{} {
	return vm.updateChan
}

func (vm *ViewModel) signalUpdate() {
	select {
	case vm.updateChan <- struct{}{}:
	default:
	}
}

// Tree building logic
func (vm *ViewModel) BuildTreeData() *TreeNode {
	top := vm.GetTopology()
	root := &TreeNode{Text: "RabbitMQ Topology"}

	vhosts := map[string]*TreeNode{}
	for _, q := range top.Queues {
		vhostNode := getOrCreateNode(vhosts, q.Vhost)
		vhostNode.Children = append(vhostNode.Children, &TreeNode{Text: "Queue: " + q.Name})
	}
	for _, ex := range top.Exchanges {
		vhostNode := getOrCreateNode(vhosts, ex.Vhost)
		vhostNode.Children = append(vhostNode.Children, &TreeNode{Text: "Exchange: " + ex.Name})
	}
	for _, node := range vhosts {
		root.Children = append(root.Children, node)
	}
	return root
}

func getOrCreateNode(vhosts map[string]*TreeNode, name string) *TreeNode {
	if vhosts[name] == nil {
		vhosts[name] = &TreeNode{Text: "VHost: " + name}
	}
	return vhosts[name]
}

// Tree diffing
func (vm *ViewModel) HasTreeChanged(newTree *TreeNode) bool {
	if !treeEqual(vm.lastTree, newTree) {
		vm.lastTree = newTree
		return true
	}
	return false
}

func treeEqual(a, b *TreeNode) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Text != b.Text || len(a.Children) != len(b.Children) {
		return false
	}
	for i := range a.Children {
		if !treeEqual(a.Children[i], b.Children[i]) {
			return false
		}
	}
	return true
}
