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

package restart_deploy

import (
	"context"
	"fmt"
	"github.com/norseto/k8s-watchdogs/pkg/k8sapps"
	"github.com/norseto/k8s-watchdogs/pkg/k8sclient"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// New returns a new Cobra command for re-balancing pods.
func New() *cobra.Command {
	var namespace, target string

	cmd := &cobra.Command{
		Use:   "restart-deploy",
		Short: "Restart deployment",
		Run: func(cmd *cobra.Command, args []string) {
			if target == "" {
				_ = cmd.Usage()
				return
			}
			_ = restartDeployment(cmd.Context(), namespace, target)
		},
	}
	flg := cmd.Flags()
	flg.StringVarP(&namespace, "namespace", "n", "default", "Namespace of target pod.")
	flg.StringVarP(&target, "target", "t", "", "The name of target deployment.")

	return cmd
}

func restartDeployment(ctx context.Context, namespace, target string) error {
	var client kubernetes.Interface

	log := logger.FromContext(ctx)

	client, err := k8sclient.NewClientset()
	if err != nil {
		log.Error(err, "failed to create client")
		return err
	}

	dep, err := client.AppsV1().Deployments(namespace).Get(ctx, target, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "failed to get deployment", "target",
			fmt.Sprintf("%s/%s", namespace, target))
		return err
	}
	if dep == nil {
		log.Error(err, "deployment not found", "target",
			fmt.Sprintf("%s/%s", namespace, target))
		return err
	}

	return k8sapps.RestartDeployment(ctx, client, dep)
}
