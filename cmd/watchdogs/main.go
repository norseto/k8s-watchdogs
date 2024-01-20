package main

import (
	"context"
	evcmd "github.com/norseto/k8s-watchdogs/internal/cmd/clean-evicted"
	docmd "github.com/norseto/k8s-watchdogs/internal/cmd/delete-oldest"
	rbcmd "github.com/norseto/k8s-watchdogs/internal/cmd/rebalance-pods"
	rdcmd "github.com/norseto/k8s-watchdogs/internal/cmd/restart-deploy"
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
		Long:  `Kubernetes utilities that can cleanup evicted pod, re-balance pod or restart deployment and so on`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Usage()
		},
	}
	rootCmd.SetContext(ctx)
	logger.InitCmdLogger(rootCmd)
	rootCmd.AddCommand(
		evcmd.New(),
		rbcmd.New(),
		docmd.New(),
		rdcmd.New(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Error(err, "Failed to execute command")
		os.Exit(1)
	}
}
