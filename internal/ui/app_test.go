package ui_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Patrick-Ivann/AIM-Q/internal/cli"
	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
	"github.com/Patrick-Ivann/AIM-Q/internal/ui"
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
func minimalTopology() *rabbitmq.Topology {
	return &rabbitmq.Topology{
		Exchanges: []rabbitmq.Exchange{{Name: "ex1", Type: "direct", Vhost: "vh1"}},
		Queues:    []rabbitmq.Queue{{Name: "q1", Vhost: "vh1"}},
		Bindings:  []rabbitmq.Binding{{Source: "ex1", Destination: "q1", DestType: "queue", Vhost: "vh1"}},
		Consumers: []rabbitmq.Consumer{{Queue: "q1", Vhost: "vh1", ConsumerTag: "consumer1", ChannelDetail: struct {
			PID int "json:\"pid\""
		}{PID: 667}}},
	}
}

// Helper to create Explorer with mock client and minimal topology.
func newExplorer() *ui.Explorer {
	mockClient := &MockClient{}
	mockClient.On("FetchTopology").Return(minimalTopology(), nil)
	explorer := ui.NewExplorer(mockClient, cli.Options{URI: "amqp://guest@localhost"}, minimalTopology(), nil)

	return explorer
}

func newExplorerWithMockClient() *ui.Explorer {
	mockClient := &MockClient{}
	mockClient.On("FetchTopology").Return(minimalTopology(), nil)
	return ui.NewExplorer(
		mockClient,
		cli.Options{URI: "amqp://guest@localhost"},
		minimalTopology(),
		nil,
	)
}

func TestStartExplorer_SuccessAndError(t *testing.T) {
	topo := minimalTopology()

	client := &MockClient{}
	client.On("FetchTopology").Return(topo, nil)

	// Successful start (don't actually .Run() the app in tests if not needed.)
	go func() {
		_ = ui.StartExplorer(client, cli.Options{}, 0, nil)
	}()

	// Simulate fetch topology error - ensure correct typed nil is returned
	clientFail := &MockClient{}
	clientFail.On("FetchTopology").Return((*rabbitmq.Topology)(nil), fmt.Errorf("fail"))
	err := ui.StartExplorer(clientFail, cli.Options{}, 0, nil)
	assert.Error(t, err)
}

func TestExplorer_InitUI_CreatesUI(t *testing.T) {
	e := newExplorer()
	e.InitUI()
	assert.NotNil(t, e.Tree)
	assert.Equal(t, " AIM-Q Topology ", e.Tree.GetTitle())
}

func TestExplorer_BuildVhostTree_Content(t *testing.T) {
	e := newExplorer()
	tree := e.BuildVhostTree()
	assert.NotNil(t, tree)
	assert.Equal(t, "RabbitMQ Vhosts", tree.GetRoot().GetText())
	children := tree.GetRoot().GetChildren()
	assert.NotEmpty(t, children)
	foundVhost := false
	for _, c := range children {
		if c.GetText() == "🐰 Vhost: vh1" {
			foundVhost = true
		}
	}
	assert.True(t, foundVhost)
}

func TestExplorer_ModalFooter_Content(t *testing.T) {
	e := newExplorer()
	tv := e.ModalFooter()
	assert.Contains(t, tv.GetText(true), "[Esc] to go back")
}

func TestExplorer_StartAutoRefresh_Run(t *testing.T) {
	e := newExplorerWithMockClient()
	mockClient := e.Client.(*MockClient)
	mockClient.On("FetchTopology").Return(minimalTopology(), nil)

	done := make(chan struct{})
	go func() {
		e.StartAutoRefresh(10 * time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)
	e.StopAutoRefresh() // Tell goroutine to stop

	// Optionally wait a small time for goroutine exit
	time.Sleep(10 * time.Millisecond)

	close(done)
	<-done
}
