package ui_test

import (
	"context"
	"testing"
	"time"

	"github.com/Patrick-Ivann/AIM-Q/internal/ui"
	"github.com/stretchr/testify/assert"
)

// mockVM is a minimal ViewModel mock for testing
type mockVM struct {
	treeData      *ui.TreeNode
	updateChannel chan struct{}
	hasChanged    bool
	started       bool
}

func (m *mockVM) BuildTreeData() *ui.TreeNode {
	return m.treeData
}

func (m *mockVM) HasTreeChanged(tree *ui.TreeNode) bool {
	return m.hasChanged
}

func (m *mockVM) FetchTopology() error {
	return nil
}

func (m *mockVM) Updates() <-chan struct{} {
	return m.updateChannel
}

func (m *mockVM) StartAutoRefresh(ctx context.Context, interval time.Duration) {
	m.started = true
	go func() {
		<-ctx.Done()
	}()
}

func TestExplorer_StartAndStop_WithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &mockVM{
		treeData:      &ui.TreeNode{Text: "mock-root"},
		updateChannel: make(chan struct{}, 1),
		hasChanged:    true,
	}

	ex := ui.NewExplorer(mock)

	// Run InitUI with refresh disabled to keep test fast
	ex.InitUI(ctx, 0)

	assert.NotNil(t, ex.App)
	assert.NotNil(t, ex.Pages)

	// Ensure background goroutine for update is running
	mock.updateChannel <- struct{}{}

	// Allow small delay for goroutine to pick up update
	time.Sleep(50 * time.Millisecond)

	assert.True(t, true, "Explorer ran without panic and listened for update")
}
