/*
MIT License

Copyright (c) 2019-2024 Norihiro Seto

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

package restartdeploy

import (
	"context"
	"fmt"
	"github.com/norseto/k8s-watchdogs/internal/options"
	"github.com/norseto/k8s-watchdogs/pkg/kube"
	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NewCommand returns a new Cobra command for re-balancing pods.
func NewCommand() *cobra.Command {
	opts := &options.Options{}
	cmd := &cobra.Command{
		Use:   "restart-deploy",
		Short: "Restart deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				_ = cmd.Usage()
				return nil
			}
			ctx := cmd.Context()
			clnt, err := client.NewClientset(client.FromContext(ctx))
			if err != nil {
				logger.FromContext(ctx).Error(err, "failed to create client")
				return err
			}
			return restartDeployment(cmd.Context(), clnt, opts.Namespace(), args)
		},
		Args: cobra.MinimumNArgs(1),
	}
	opts.BindCommonFlags(cmd)

	return cmd
}

func restartDeployment(ctx context.Context, client kubernetes.Interface, namespace string, targets []string) error {
	log := logger.FromContext(ctx)

	for _, target := range targets {
		dep, err := client.AppsV1().Deployments(namespace).Get(ctx, target, metav1.GetOptions{})
		if err != nil || dep == nil {
			log.Error(err, "failed to get deployment", "target",
				fmt.Sprintf("%s/%s", namespace, target))
			return err
		}

		err = kube.RestartDeployment(ctx, client, dep)
		if err != nil {
			log.Error(err, "failed to restart deployment", "target",
				fmt.Sprintf("%s/%s", namespace, target))
			return err
		}
	}
	return nil
}
