package ui_test

import (
	"testing"

	"github.com/Patrick-Ivann/AIM-Q/internal/ui"
	"github.com/stretchr/testify/assert"
)

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
