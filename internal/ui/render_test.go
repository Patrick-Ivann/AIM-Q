package ui_test

import (
	"testing"

	"github.com/Patrick-Ivann/AIM-Q/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestRenderTree(t *testing.T) {
	root := &ui.TreeNode{
		Text: "root",
		Children: []*ui.TreeNode{
			{Text: "child1"},
			{Text: "child2", Children: []*ui.TreeNode{
				{Text: "grandchild"},
			}},
		},
	}

	tviewNode := ui.RenderTree(root)

	assert.NotNil(t, tviewNode)
	assert.Equal(t, "root", tviewNode.GetText())
	children := tviewNode.GetChildren()
	assert.Len(t, children, 2)
	assert.Equal(t, "child1", children[0].GetText())
	assert.Equal(t, "child2", children[1].GetText())

	grandchildren := children[1].GetChildren()
	assert.Len(t, grandchildren, 1)
	assert.Equal(t, "grandchild", grandchildren[0].GetText())
}
func TestRenderTreeWhenNoData(t *testing.T) {

	tviewNode := ui.RenderTree(nil)

	assert.Nil(t, tviewNode)
}
