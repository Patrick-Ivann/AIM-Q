package ui_test

import (
	"testing"

	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
	"github.com/Patrick-Ivann/AIM-Q/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock rabbitmq.Client supporting FetchTopology call.
type MockClient struct {
	mock.Mock
}

func (m *MockClient) FetchTopology() (*rabbitmq.Topology, error) {
	args := m.Called()
	return args.Get(0).(*rabbitmq.Topology), args.Error(1)
}

func (m *MockClient) Get(path string, out interface{}) error {
	args := m.Called(path, out)
	return args.Error(0)
}

// Provide a minimal valid topology for tests.
func MinimalTopology() *rabbitmq.Topology {
	return &rabbitmq.Topology{
		Exchanges: []rabbitmq.Exchange{{Name: "ex1", Type: "direct", Vhost: "vh1"}, {Name: "ex1", Type: "topic", Vhost: "/"}},
		Queues:    []rabbitmq.Queue{{Name: "q1", Vhost: "vh1"}},
		Bindings:  []rabbitmq.Binding{{Source: "ex1", Destination: "q1", DestType: "queue", Vhost: "vh1"}},
		Consumers: []rabbitmq.Consumer{{Queue: "q1", Vhost: "vh1", ConsumerTag: "consumer1", ChannelDetail: struct {
			PID int "json:\"pid\""
		}{PID: 667}}},
	}
}

func TestBuildTreeData(t *testing.T) {
	mockClient := &MockClient{}
	mockClient.On("FetchTopology").Return(MinimalTopology(), nil)

	vm := ui.NewViewModel(mockClient)

	err := vm.FetchTopology()
	assert.NoError(t, err)

	tree := vm.BuildTreeData()
	assert.Equal(t, "RabbitMQ Topology", tree.Text)
	assert.Len(t, tree.Children, 2)
	vhost := tree.Children[0]
	vhost2 := tree.Children[1]

	assert.Contains(t, vhost.Text+" "+vhost2.Text, "VHost: /")
}

func TestHasTreeChanged(t *testing.T) {
	vm := &ui.ViewModel{}
	tree1 := &ui.TreeNode{Text: "root"}
	tree2 := &ui.TreeNode{Text: "root"}
	tree3 := &ui.TreeNode{Text: "changed"}

	assert.True(t, vm.HasTreeChanged(tree1))  // first time should be true
	assert.False(t, vm.HasTreeChanged(tree2)) // same tree, should be false
	assert.True(t, vm.HasTreeChanged(tree3))  // changed tree, should be true
}
