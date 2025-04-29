/*
MIT License

Copyright (c) 2025 Norihiro Seto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"context"
	"os"

	watchdogs "github.com/norseto/k8s-watchdogs"
	cecmd "github.com/norseto/k8s-watchdogs/internal/cmd/clean-evicted"
	docmd "github.com/norseto/k8s-watchdogs/internal/cmd/delete-oldest"
	rpcmd "github.com/norseto/k8s-watchdogs/internal/cmd/rebalance-pods"
	rdcmd "github.com/norseto/k8s-watchdogs/internal/cmd/restart-deploy"
	rscmd "github.com/norseto/k8s-watchdogs/internal/cmd/restart-sts"
	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/spf13/cobra"
)

func main() {
	opts := &client.Options{}
	ctx := client.WithContext(context.Background(), opts)

	var rootCmd = &cobra.Command{
		Use:   "watchdogs",
		Short: "Kubernetes watchdogs utilities",
		Long:  `Kubernetes utilities that can cleanup evicted pod, re-balance pod or restart deployment and so on`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Usage()
		},
	}
	rootCmd.SetContext(ctx)
	logger.InitCmdLogger(rootCmd, func(cmd *cobra.Command, args []string) {
		logger.FromContext(cmd.Context()).Info("Starting watchdogs", "version", watchdogs.RELEASE_VERSION, "GitVersion", watchdogs.GitVersion)
	})
	opts.BindPFlags(rootCmd.PersistentFlags())
	rootCmd.AddCommand(
		cecmd.NewCommand(),
		rpcmd.NewCommand(),
		docmd.NewCommand(),
		rdcmd.NewCommand(),
		rscmd.NewCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		logger.FromContext(ctx).Error(err, "Failed to execute command")
		os.Exit(1)
	}
}
