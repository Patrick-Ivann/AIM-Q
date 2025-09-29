package ui

import (
	"github.com/rivo/tview"
)

// RenderTree converts a TreeNode (pure data) to tview.TreeNode
func RenderTree(data *TreeNode) *tview.TreeNode {
	if data == nil {
		return nil
	}
	node := tview.NewTreeNode(data.Text)
	for _, child := range data.Children {
		node.AddChild(RenderTree(child))
	}
	return node
}
