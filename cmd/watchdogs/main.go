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
	vrcmd "github.com/norseto/k8s-watchdogs/internal/cmd/version"
	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	rootCmdFactory = NewRootCmd
	exitFunc       = os.Exit
)

func main() {
	exitFunc(run())
}

// run executes the root command and returns an exit code instead of exiting directly.
// This indirection keeps main() minimal while allowing unit tests to exercise the
// success and failure paths without terminating the test process.
func run() int {
	rootCmd := rootCmdFactory()
	if err := rootCmd.Execute(); err != nil {
		logger.FromContext(rootCmd.Context()).Error(err, "Failed to execute command")
		return 1
	}
	return 0
}

// NewRootCmd creates and returns the root cobra.Command for the watchdogs CLI application.
// It sets up the command structure, context, logging, and persistent flags.
func NewRootCmd() *cobra.Command {
	opts := &client.Options{}
	ctx := client.WithContext(context.Background(), opts)

	rootCmd := &cobra.Command{
		Use:   "watchdogs",
		Short: "Kubernetes watchdogs utilities",
		Long:  `Kubernetes utilities that can cleanup evicted pod, re-balance pod or restart deployment and so on`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Usage()
		},
	}
	rootCmd.SetContext(ctx)
	logger.InitCmdLogger(rootCmd, func(cmd *cobra.Command, args []string) {
		logger.FromContext(cmd.Context()).Info("Starting watchdogs",
			"version", watchdogs.RELEASE_VERSION, "GitVersion", watchdogs.GitVersion)
	})
	opts.BindPFlags(rootCmd.PersistentFlags())
	rootCmd.AddCommand(
		cecmd.NewCommand(),
		rpcmd.NewCommand(),
		docmd.NewCommand(),
		rdcmd.NewCommand(),
		rscmd.NewCommand(),
		vrcmd.NewCommand(),
	)

	return rootCmd
}
