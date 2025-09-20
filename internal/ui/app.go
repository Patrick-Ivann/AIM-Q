package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/Patrick-Ivann/AIM-Q/internal/cli"
	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Explorer represents the TUI application.
type Explorer struct {
	App      *tview.Application
	Pages    *tview.Pages
	Client   *rabbitmq.Client
	Opts     cli.Options
	Topology *rabbitmq.Topology
	Tree     *tview.TreeView

	refreshMu sync.Mutex

	SelectedNodePath string
	ExpandedNodes    map[string]bool
}

// StartExplorer runs the TUI app with the given topology.
func StartExplorer(client *rabbitmq.Client, opts cli.Options, refreshInterval time.Duration) error {
	topology, err := client.FetchTopology()
	if err != nil {
		return err
	}

	explorer := &Explorer{
		App:      tview.NewApplication(),
		Pages:    tview.NewPages(),
		Client:   client,
		Opts:     opts,
		Topology: topology,
	}

	explorer.initUI()

	// Start auto-refresh loop
	if refreshInterval > 0 {
		go explorer.startAutoRefresh(refreshInterval)
	}

	return explorer.App.SetRoot(explorer.Pages, true).Run()
}

func (e *Explorer) initUI() {
	e.Tree = e.buildVhostTree()
	e.Tree.SetBorder(true).SetTitle(" AIM-Q Topology ").SetTitleAlign(tview.AlignLeft)

	helpText := tview.NewTextView().
		SetText("Arrow keys to navigate, [Enter] to select, [Esc] to quit").
		SetTextColor(tcell.ColorGray)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(e.Tree, 0, 1, true).
		AddItem(helpText, 1, 1, false)

	e.Pages.AddPage("main", layout, true, true)
}

func (e *Explorer) buildVhostTree() *tview.TreeView {
	root := tview.NewTreeNode("RabbitMQ Vhosts")
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)

	vhostMap := map[string]*tview.TreeNode{}

	for _, ex := range e.Topology.Exchanges {
		if _, ok := vhostMap[ex.Vhost]; !ok {
			vhostNode := tview.NewTreeNode(fmt.Sprintf("🐰 Vhost: %s", ex.Vhost)).SetExpanded(false)
			vhostMap[ex.Vhost] = vhostNode
			root.AddChild(vhostNode)
		}
		exNode := tview.NewTreeNode(fmt.Sprintf("🔁 Exchange: %s (%s)", ex.Name, ex.Type)).
			SetReference(&ex).
			SetExpanded(false).
			SetSelectable(true)
		vhostMap[ex.Vhost].AddChild(exNode)
	}

	for _, q := range e.Topology.Queues {
		if _, ok := vhostMap[q.Vhost]; !ok {
			vhostNode := tview.NewTreeNode(fmt.Sprintf("🐰 Vhost: %s", q.Vhost)).SetExpanded(false)
			vhostMap[q.Vhost] = vhostNode
			root.AddChild(vhostNode)
		}
		qNode := tview.NewTreeNode(fmt.Sprintf("📦 Queue: %s", q.Name)).
			SetReference(&q).
			SetExpanded(false).
			SetSelectable(true)
		vhostMap[q.Vhost].AddChild(qNode)
	}

	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			node.SetExpanded(!node.IsExpanded()) // toggle expansion for vhost node
			return
		}

		switch val := ref.(type) {
		case *rabbitmq.Exchange:
			e.showExchangeDetails(val)
		case *rabbitmq.Queue:
			e.showQueueDetails(val)
		}
	})

	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			e.App.Stop()
			return nil
		}
		return event
	})

	tree.SetCurrentNode(root)

	return tree
}

func (e *Explorer) showExchangeDetails(ex *rabbitmq.Exchange) {
	text := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	fmt.Fprintf(text, "[::b]Exchange:[-:-] %s\n", ex.Name)
	fmt.Fprintf(text, "Type: %s\nDurable: %v\nAuto-Delete: %v\n", ex.Type, ex.Durable, ex.AutoDelete)

	fmt.Fprintln(text, "\nBindings:")
	for _, b := range e.Topology.Bindings {
		if b.Source == ex.Name && b.Vhost == ex.Vhost {
			count := "" // or derive from queue stats
			fmt.Fprintf(text, "  ➤ %s → %s (%s) [key: %s]%s\n",
				b.Source, b.Destination, b.DestType, b.RoutingKey, count)
		}
	}

	text.SetBorder(true).SetTitle(fmt.Sprintf(" Exchange: %s ", ex.Name))

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(text, 0, 1, true).
		AddItem(e.modalFooter(), 1, 0, false)

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			e.Pages.SwitchToPage("main")
			return nil
		}
		return event
	})

	e.Pages.AddAndSwitchToPage("details", modal, true)
	e.App.SetFocus(text)
}

func (e *Explorer) showQueueDetails(q *rabbitmq.Queue) {
	text := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)

	fmt.Fprintf(text, "[::b]Queue:[-:-] %s\n", q.Name)
	fmt.Fprintf(text, "Durable: %v\nAuto-Delete: %v\n", q.Durable, q.AutoDelete)
	if q.MessageStats.MessagesReady != 0 {
		fmt.Fprintf(text, "Ready messages: %v\n", q.MessageStats.MessagesReady)
	}
	if q.MessageStats.MessagesUnacked != 0 {
		fmt.Fprintf(text, "Unacknowledged messages: %v\n", q.MessageStats.MessagesUnacked)
	}

	// Bindings to this queue
	fmt.Fprintln(text, "\nBindings:")
	for _, b := range e.Topology.Bindings {
		if b.Destination == q.Name && b.Vhost == q.Vhost && b.DestType == "queue" {
			fmt.Fprintf(text, "  ➤ %s → %s [key: %s]\n", b.Source, b.Destination, b.RoutingKey)
		}
	}

	// Consumers
	fmt.Fprintln(text, "\nConsumers:")
	for _, c := range e.Topology.Consumers {
		if c.Queue == q.Name && c.Vhost == q.Vhost {
			fmt.Fprintf(text, "  ➤ %s (PID %d)\n", c.ConsumerTag, c.ChannelDetail.PID)
		}
	}

	text.SetBorder(true).SetTitle(fmt.Sprintf(" Queue: %s ", q.Name))

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(text, 0, 1, true).
		AddItem(e.modalFooter(), 1, 0, false)

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			e.Pages.SwitchToPage("main")
			return nil
		}
		return event
	})

	e.Pages.AddAndSwitchToPage("details", modal, true)
	e.App.SetFocus(text)
}

func (e *Explorer) startAutoRefresh(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		newTopo, err := e.Client.FetchTopology()
		if err != nil {
			continue
		}

		e.refreshMu.Lock()
		e.Topology = newTopo
		e.refreshTreeView()
		e.refreshMu.Unlock()
	}
}

func (e *Explorer) refreshTreeView() {
	// Capture current selection and expanded nodes before refreshing
	var selectedText string
	treeView, ok := e.App.GetFocus().(*tview.TreeView)
	if ok && treeView != nil {
		if current := treeView.GetCurrentNode(); current != nil {
			selectedText = current.GetText()
		}
		e.ExpandedNodes = collectExpandedNodes(treeView.GetRoot())
	}

	// Fetch new topology from RabbitMQ
	client, clientErr := rabbitmq.NewClient(e.Opts.URI)
	if clientErr != nil {
		return
	}
	newTopology, err := client.FetchTopology()
	if err != nil {
		// You might want to log or display this error in the TUI
		return
	}
	e.Topology = newTopology

	// Rebuild the tree and layout
	newTree := e.buildVhostTree()
	e.restoreTreeState(newTree, selectedText)

	// Help text
	helpText := tview.NewTextView().
		SetText("Arrow keys to navigate, [Enter] to select, [Esc] to quit").
		SetTextColor(tcell.ColorGray)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(newTree, 0, 1, true).
		AddItem(helpText, 1, 1, false)

	// Replace the page in a safe way
	e.App.QueueUpdateDraw(func() {
		e.Pages.RemovePage("main")
		e.Pages.AddPage("main", layout, true, true)
	})
}

func (e *Explorer) restoreTreeState(tree *tview.TreeView, selectedText string) {
	var walk func(node *tview.TreeNode)
	walk = func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		text := node.GetText()

		if e.ExpandedNodes != nil && e.ExpandedNodes[text] {
			node.SetExpanded(true)
		}
		if text == selectedText {
			tree.SetCurrentNode(node)
		}

		for _, child := range node.GetChildren() {
			walk(child)
		}
	}

	walk(tree.GetRoot())
}

func collectExpandedNodes(root *tview.TreeNode) map[string]bool {
	expanded := make(map[string]bool)
	var walk func(node *tview.TreeNode)
	walk = func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		expanded[node.GetText()] = node.IsExpanded()
		for _, child := range node.GetChildren() {
			walk(child)
		}
	}
	walk(root)
	return expanded
}

func (e *Explorer) modalFooter() *tview.TextView {
	footer := tview.NewTextView().
		SetText("[Esc] to go back").
		SetTextAlign(tview.AlignRight).
		SetTextColor(tcell.ColorGray)
	return footer
}
