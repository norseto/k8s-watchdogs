/*
MIT License

Copyright (c) 2024 Norihiro Seto

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
