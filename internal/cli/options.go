package cli


// Options contains command line arguments passed to generate or tui commands.
type Options struct {
	URI            string
	GroupBy        string
	FilterVhost    string
	FilterExchange string
	OutFile        string
	ShowMsgStats   bool
}