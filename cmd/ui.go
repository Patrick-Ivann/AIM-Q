package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Patrick-Ivann/AIM-Q/internal/cli"
	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
	"github.com/Patrick-Ivann/AIM-Q/internal/ui"
	"github.com/spf13/cobra"
)

var (
	refreshInterval int
	opts            cli.Options
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start interactive TUI for exploring RabbitMQ topology",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Context and signal handling
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			log.Println("Received SIGINT, shutting down TUI...")
			cancel()
		}()

		httpClient := http.DefaultClient
		client, err := rabbitmq.NewClient(uri, httpClient)
		if err != nil {
			return err
		}
		vm := ui.NewViewModel(client)
		explorer := ui.NewExplorer(vm)
		return explorer.Start(ctx, 5*time.Second)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)

	tuiCmd.Flags().StringVarP(&uri, "uri", "u", "http://guest:guest@localhost:15672", "RabbitMQ management URI")
	tuiCmd.Flags().IntVar(&refreshInterval, "refresh-interval", 10, "TUI refresh interval in seconds")

	// Same filtering options as generate
	tuiCmd.Flags().StringVar(&opts.FilterVhost, "filter-vhost", "", "Only include objects from this vhost")
	tuiCmd.Flags().StringVar(&opts.FilterExchange, "filter-exchange", "", "Only include objects from this exchange")
	tuiCmd.Flags().StringVar(&opts.GroupBy, "group-by", "vhost", "Group diagram by 'vhost' or 'type'")
	tuiCmd.Flags().BoolVar(&opts.ShowMsgStats, "message-stats", false, "Show message stats (if available)")
}
