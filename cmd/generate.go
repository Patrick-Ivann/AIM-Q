package cmd

import (
	"fmt"
	"os"

	"github.com/Patrick-Ivann/AIM-Q/internal/cli"
	"github.com/Patrick-Ivann/AIM-Q/internal/diagram"
	"github.com/Patrick-Ivann/AIM-Q/internal/logger"
	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
	"github.com/spf13/cobra"
)

var (
	uri            string
	groupBy        string
	filterVhost    string
	filterExchange string
	outFile        string
	showMsgStats   bool
)

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVar(&uri, "uri", "", "RabbitMQ Management URI (e.g. http://user:pass@localhost:15672)")
	generateCmd.Flags().StringVar(&groupBy, "group-by", "vhost", "Group output by (vhost/type)")
	generateCmd.Flags().StringVar(&filterVhost, "filter-vhost", "", "Filter by virtual host")
	generateCmd.Flags().StringVar(&filterExchange, "filter-exchange", "", "Filter by exchange name")
	generateCmd.Flags().BoolVar(&showMsgStats, "message-stats", false, "Include message statistics in output")
	generateCmd.Flags().StringVar(&outFile, "out", "topology.puml", "Output file path")
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate PlantUML topology from RabbitMQ",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.New()

		if uri == "" {
			return fmt.Errorf("missing --uri")
		}

		log.Info("connecting to RabbitMQ at: %s", "uri", uri)

		opts := cli.Options{
			URI:            uri,
			GroupBy:        groupBy,
			FilterVhost:    filterVhost,
			FilterExchange: filterExchange,
			OutFile:        outFile,
			ShowMsgStats:   showMsgStats,
		}

		client, clientErr := rabbitmq.NewClient(opts.URI)

		if clientErr != nil {
			return fmt.Errorf("connection error to broker : %w", clientErr)
		}

		topology, err := client.FetchTopology()
		if err != nil {
			return fmt.Errorf("fetch error: %w", err)
		}

		topology = topology.Filter(opts)

		plantuml := diagram.Generate(topology, opts)

		if err := os.WriteFile(opts.OutFile, []byte(plantuml), 0644); err != nil {
			return fmt.Errorf("writing file failed: %w", err)
		}
		log.Info("✅ Output written", "path", opts.OutFile)
		return nil
	},
}
