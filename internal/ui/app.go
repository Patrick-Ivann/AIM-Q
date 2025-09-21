// Package ui provides a TUI explorer for RabbitMQ topology.
package ui

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Patrick-Ivann/AIM-Q/internal/cli"
	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Explorer is the main struct representing the interactive TUI application
// for visualizing and exploring RabbitMQ topology.
//
// Explorer manages rendering, user navigation, auto-refresh, and detail popups.
// Fields with uppercase initials are exported for testing or external use.
//
// Only one Explorer should be live per TUI session.
type Explorer struct {
	App              *tview.Application       // TUI Application instance.
	Pages            *tview.Pages             // Manages layered views/pages.
	Client           rabbitmq.ClientInterface // Used for querying RabbitMQ data.
	Opts             cli.Options              // Command-line options, typically including URI.
	Topology         *rabbitmq.Topology       // Latest snapshot of the full MQ topology graph.
	Tree             *tview.TreeView          // Main navigation tree for vhosts/exchanges/queues.
	HttpClient       *rabbitmq.HTTPClient
	refreshMu        sync.Mutex      // Guards reload during auto-refresh.
	SelectedNodePath string          // Path of current navigation selection.
	ExpandedNodes    map[string]bool // Maps expanded node texts for sync after refresh.
	stopAutoRefresh  chan struct{}   // channel to signal auto-refresh to stop
}

// StartExplorer launches the TUI application given a RabbitMQ client and CLI options.
// Returns a non-nil error if the initial topology fetch fails or on TUI exit/failure.
//
// If refreshInterval > 0, live topology refresh is enabled via background goroutine.
func StartExplorer(client rabbitmq.ClientInterface, opts cli.Options, refreshInterval time.Duration, httpClient *rabbitmq.HTTPClient) error {
	topology, err := client.FetchTopology()
	if err != nil {
		return err
	}

	explorer := NewExplorer(client, opts, topology, httpClient)
	explorer.InitUI()

	// Enable background auto-refresh of topology if requested.
	if refreshInterval > 0 {
		go explorer.StartAutoRefresh(refreshInterval)
	}

	return explorer.App.SetRoot(explorer.Pages, true).Run()
}

// NewExplorer allocates and returns a new Explorer instance.
func NewExplorer(client rabbitmq.ClientInterface, opts cli.Options, topology *rabbitmq.Topology, httpClient *rabbitmq.HTTPClient) *Explorer {
	return &Explorer{
		App:             tview.NewApplication(),
		Pages:           tview.NewPages(),
		Client:          client,
		Opts:            opts,
		Topology:        topology,
		HttpClient:      httpClient,
		stopAutoRefresh: make(chan struct{}, 1),
	}
}

// initUI configures the main navigation and help widgets for the TUI.
func (e *Explorer) InitUI() {
	e.Tree = e.BuildVhostTree()
	e.Tree.SetBorder(true).
		SetTitle(" AIM-Q Topology ").
		SetTitleAlign(tview.AlignLeft)

	// Help text displayed at the bottom of the UI window.
	helpText := tview.NewTextView().
		SetText("Arrow keys to navigate, [Enter] to select, [Esc] to quit").
		SetTextColor(tcell.ColorGray)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(e.Tree, 0, 1, true).
		AddItem(helpText, 1, 1, false)

	e.Pages.AddPage("main", layout, true, true)
}

// BuildVhostTree constructs the hierarchical navigation tree for vhosts,
// exchanges, and queues mapping the topology fields. Returns a fully
// configured TreeView, ready for event handlers.
func (e *Explorer) BuildVhostTree() *tview.TreeView {
	root := tview.NewTreeNode("RabbitMQ Vhosts")
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	// vhostMap avoids duplicate vhost nodes and allows O(1) lookup.
	vhostMap := e.createVhostNodes(root)
	e.addExchangeNodes(vhostMap)
	e.addQueueNodes(vhostMap)

	e.setupTreeHandlers(tree, root)
	return tree
}

// createVhostNodes scans the current exchanges and queues to populate top-level vhost nodes.
// Returns a map from vhost name to their TreeNode, allowing children insertion.
func (e *Explorer) createVhostNodes(root *tview.TreeNode) map[string]*tview.TreeNode {
	vhostMap := make(map[string]*tview.TreeNode)
	for _, ex := range e.Topology.Exchanges {
		if _, ok := vhostMap[ex.Vhost]; !ok {
			vhostNode := tview.NewTreeNode(fmt.Sprintf("🐰 Vhost: %s", ex.Vhost)).SetExpanded(false)
			vhostMap[ex.Vhost] = vhostNode
			root.AddChild(vhostNode)
		}
	}
	for _, q := range e.Topology.Queues {
		if _, ok := vhostMap[q.Vhost]; !ok {
			vhostNode := tview.NewTreeNode(fmt.Sprintf("🐰 Vhost: %s", q.Vhost)).SetExpanded(false)
			vhostMap[q.Vhost] = vhostNode
			root.AddChild(vhostNode)
		}
	}
	return vhostMap
}

// addExchangeNodes inserts Exchange children beneath their parent vhost nodes.
func (e *Explorer) addExchangeNodes(vhostMap map[string]*tview.TreeNode) {
	for _, ex := range e.Topology.Exchanges {
		exNode := tview.NewTreeNode(fmt.Sprintf("🔁 Exchange: %s (%s)", ex.Name, ex.Type)).
			SetReference(&ex).SetExpanded(false).SetSelectable(true)
		vhostMap[ex.Vhost].AddChild(exNode)
	}
}

// addQueueNodes inserts Queue children beneath their parent vhost nodes.
func (e *Explorer) addQueueNodes(vhostMap map[string]*tview.TreeNode) {
	for _, q := range e.Topology.Queues {
		qNode := tview.NewTreeNode(fmt.Sprintf("📦 Queue: %s", q.Name)).
			SetReference(&q).SetExpanded(false).SetSelectable(true)
		vhostMap[q.Vhost].AddChild(qNode)
	}
}

// setupTreeHandlers wires navigation and keyboard input handlers for the tree.
func (e *Explorer) setupTreeHandlers(tree *tview.TreeView, root *tview.TreeNode) {
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			// Vhost node: toggle expansion/collapse.
			node.SetExpanded(!node.IsExpanded())
			return
		}
		// Exchange or Queue node: open details view.
		switch val := ref.(type) {
		case *rabbitmq.Exchange:
			e.showExchangeDetails(val)
		case *rabbitmq.Queue:
			e.showQueueDetails(val)
		}
	})

	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			e.App.Stop()
			return nil
		}
		return event
	})

	tree.SetCurrentNode(root)
}

// showExchangeDetails shows a modal with full Exchange details.
func (e *Explorer) showExchangeDetails(ex *rabbitmq.Exchange) {
	text := e.formatExchangeDetails(ex)
	modal := e.buildModal(text)

	// Modal closes with Esc, returns to main page.
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

// formatExchangeDetails produces a text description for the Exchange, including its bindings.
func (e *Explorer) formatExchangeDetails(ex *rabbitmq.Exchange) *tview.TextView {
	text := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	fmt.Fprintf(text, "[::b]Exchange:[-:-] %s\n", ex.Name)
	fmt.Fprintf(text, "Type: %s\nDurable: %v\nAuto-Delete: %v\n", ex.Type, ex.Durable, ex.AutoDelete)

	fmt.Fprintln(text, "\nBindings:")
	for _, b := range e.Topology.Bindings {
		if b.Source == ex.Name && b.Vhost == ex.Vhost {
			count := "" // Add live queue/stat integration here.
			fmt.Fprintf(text, "  ➤ %s → %s (%s) [key: %s]%s\n",
				b.Source, b.Destination, b.DestType, b.RoutingKey, count)
		}
	}

	text.SetBorder(true).SetTitle(fmt.Sprintf(" Exchange: %s ", ex.Name))
	return text
}

// showQueueDetails visualizes all details and bindings for a queue node.
func (e *Explorer) showQueueDetails(q *rabbitmq.Queue) {
	text := e.formatQueueDetails(q)
	modal := e.buildModal(text)

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

// formatQueueDetails produces a text description of Queue stats, bindings, and consumers.
func (e *Explorer) formatQueueDetails(q *rabbitmq.Queue) *tview.TextView {
	text := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)

	fmt.Fprintf(text, "[::b]Queue:[-:-] %s\n", q.Name)
	fmt.Fprintf(text, "Durable: %v\nAuto-Delete: %v\n", q.Durable, q.AutoDelete)
	if q.MessageStats.MessagesReady != 0 {
		fmt.Fprintf(text, "Ready messages: %v\n", q.MessageStats.MessagesReady)
	}
	if q.MessageStats.MessagesUnacked != 0 {
		fmt.Fprintf(text, "Unacknowledged messages: %v\n", q.MessageStats.MessagesUnacked)
	}

	// List all queue bindings for this queue.
	fmt.Fprintln(text, "\nBindings:")
	for _, b := range e.Topology.Bindings {
		if b.Destination == q.Name && b.Vhost == q.Vhost && b.DestType == "queue" {
			fmt.Fprintf(text, "  ➤ %s → %s [key: %s]\n", b.Source, b.Destination, b.RoutingKey)
		}
	}

	// Show all consumers for this queue.
	fmt.Fprintln(text, "\nConsumers:")
	for _, c := range e.Topology.Consumers {
		if c.Queue == q.Name && c.Vhost == q.Vhost {
			fmt.Fprintf(text, "  ➤ %s (PID %d)\n", c.ConsumerTag, c.ChannelDetail.PID)
		}
	}

	text.SetBorder(true).SetTitle(fmt.Sprintf(" Queue: %s ", q.Name))
	return text
}

// buildModal constructs a modal-flex layout for details popups.
func (e *Explorer) buildModal(content *tview.TextView) *tview.Flex {
	return tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true).
		AddItem(e.ModalFooter(), 1, 0, false)
}

// StartAutoRefresh runs periodic topology refresh and updates the UI.
func (e *Explorer) StartAutoRefresh(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			newTopo, err := e.Client.FetchTopology()
			if err != nil {
				log.Printf("auto-refresh error: %v", err)
				continue
			}
			e.refreshMu.Lock()
			e.Topology = newTopo
			e.RefreshTreeView()
			e.refreshMu.Unlock()

		case <-e.stopAutoRefresh:
			return
		}
	}
}

// RefreshTreeView safely replaces the vhost tree after a topology update.
// Current navigation position and expanded nodes are preserved.
func (e *Explorer) RefreshTreeView() {
	var selectedText string
	treeView, ok := e.App.GetFocus().(*tview.TreeView)
	if ok && treeView != nil && treeView.GetCurrentNode() != nil {
		selectedText = treeView.GetCurrentNode().GetText()
		e.ExpandedNodes = collectExpandedNodes(treeView.GetRoot())
	}

	newTopology, err := e.Client.FetchTopology()
	if err != nil {
		log.Printf("topology error: %v", err)
		return
	}
	e.Topology = newTopology

	newTree := e.BuildVhostTree()
	e.restoreTreeState(newTree, selectedText)

	helpText := tview.NewTextView().
		SetText("Arrow keys to navigate, [Enter] to select, [Esc] to quit").
		SetTextColor(tcell.ColorGray)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(newTree, 0, 1, true).
		AddItem(helpText, 1, 1, false)

	// Replace the 'main' page atomically so users never see flicker or partial redraw.
	e.App.QueueUpdateDraw(func() {
		e.Pages.RemovePage("main")
		e.Pages.AddPage("main", layout, true, true)
	})
}

// restoreTreeState re-expands tree nodes and restores selection after a UI refresh.
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

// collectExpandedNodes recurses the tree and saves the expanded state of all nodes.
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

// ModalFooter provides a consistent UI footer for modal dialogs.
func (e *Explorer) ModalFooter() *tview.TextView {
	return tview.NewTextView().
		SetText("[Esc] to go back").
		SetTextAlign(tview.AlignRight).
		SetTextColor(tcell.ColorGray)
}

func (e *Explorer) StopAutoRefresh() {
	select {
	case e.stopAutoRefresh <- struct{}{}:
	default:
	}
}
