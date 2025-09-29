package ui

import (
	"context"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Explorer struct {
	App       *tview.Application
	Pages     *tview.Pages
	Help      *tview.TextView
	VM        ViewModelInterface
	cancelCtx context.CancelFunc
}

func NewExplorer(vm ViewModelInterface) *Explorer {
	return &Explorer{
		App:   tview.NewApplication(),
		Pages: tview.NewPages(),
		Help:  buildHelpView(),
		VM:    vm,
	}
}

// InitUI sets up the layout, starts listening for updates, and optionally starts auto-refresh.
func (e *Explorer) InitUI(ctx context.Context, refreshInterval time.Duration) {
	tree := e.BuildTreeFromVM()

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tree, 0, 1, true).
		AddItem(e.Help, 1, 1, false)

	e.Pages.AddPage("main", layout, true, true)

	// Start listening to ViewModel updates
	go e.listenForUpdates()

	// Start auto-refresh if interval > 0
	if refreshInterval > 0 {
		e.VM.StartAutoRefresh(ctx, refreshInterval)
	}
	// Stop the app cleanly when context is cancelled
	go func() {
		<-ctx.Done()
		e.App.QueueUpdateDraw(func() {
			e.App.Stop()
		})
	}()
}

func (e *Explorer) BuildTreeFromVM() *tview.TreeView {
	dataTree := e.VM.BuildTreeData()
	rootNode := RenderTree(dataTree)
	return tview.NewTreeView().SetRoot(rootNode).SetCurrentNode(rootNode)
}

// Start launches the UI with initial data load and refresh.
func (e *Explorer) Start(ctx context.Context, refreshInterval time.Duration) error {
	// Save cancel func if caller gave us a cancellable context
	if cancelFunc := contextCancelFunc(ctx); cancelFunc != nil {
		e.cancelCtx = cancelFunc
	}

	// Initial fetch
	if err := e.VM.FetchTopology(); err != nil {
		return err
	}

	e.InitUI(ctx, refreshInterval)
	return e.App.SetRoot(e.Pages, true).Run()
}

func (e *Explorer) Stop() {
	if e.cancelCtx != nil {
		e.cancelCtx()
	}
}

// internal: safely get CancelFunc from context (if it exists)
func contextCancelFunc(ctx context.Context) context.CancelFunc {
	type canceler interface {
		Cancel(error)
	}
	if c, ok := ctx.(interface {
		Done() <-chan struct{}
		Cancel()
	}); ok {
		return c.Cancel
	}
	return nil
}

func (e *Explorer) listenForUpdates() {
	for range e.VM.Updates() {
		dataTree := e.VM.BuildTreeData()
		if !e.VM.HasTreeChanged(dataTree) {
			continue
		}

		treeView := tview.NewTreeView().SetRoot(RenderTree(dataTree))
		layout := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(treeView, 0, 1, true).
			AddItem(e.Help, 1, 1, false)

		e.App.QueueUpdateDraw(func() {
			e.Pages.RemovePage("main")
			e.Pages.AddPage("main", layout, true, true)
		})
	}
}

func buildHelpView() *tview.TextView {
	return tview.NewTextView().
		SetText("Arrow keys to navigate, [Enter] to select, [Esc] to quit").
		SetTextColor(tcell.ColorGray)
}
