package main

import (
	"context"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	ctx := logger.WithContext(context.Background(), logger.InitLogger())
	log := logger.FromContext(ctx, "cmd", "k8s-watchdogs")

	var rootCmd = &cobra.Command{
		Use:   "watchdogs",
		Short: "Kubernetes watchdogs utilities",
		Long:  `Kubernetes utilities that can cleanup evicted pod, rebalance pod or restart deployment and so on`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Usage()
		},
	}
	rootCmd.AddCommand(
		NewCleanEvictedCmd(),
	)
	logger.SetCmdContext(ctx, rootCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Error(err, "Failed to execute command")
		os.Exit(1)
	}
}
