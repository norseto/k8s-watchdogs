package main

import (
	"context"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	ctx := context.Background()

	var rootCmd = &cobra.Command{
		Use:   "watchdogs",
		Short: "Kubernetes watchdogs utilities",
		Long:  `Kubernetes utilities that can cleanup evicted pod, rebalance pod or restart deployment and so on`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Usage()
		},
	}
	rootCmd.SetContext(ctx)
	logger.InitLogger(rootCmd)
	rootCmd.AddCommand(
		NewCleanEvictedCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Error(err, "Failed to execute command")
		os.Exit(1)
	}
}
