package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "AIM-Q",
	Short: "AIM-Q generates PlantUML diagrams of RabbitMQ topologies",
	Long: `AIM-Q connects to RabbitMQ management API and renders exchanges,
bindings, queues and consumers in a visual PlantUML format.`,
}

func Execute() error {
	return rootCmd.Execute()
}
