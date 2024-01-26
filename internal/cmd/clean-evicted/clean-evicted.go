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

package cleanevicted

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"

	"github.com/norseto/k8s-watchdogs/internal/options"
	"github.com/norseto/k8s-watchdogs/pkg/k8sclient"
	"github.com/norseto/k8s-watchdogs/pkg/k8score"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCommand returns a new Cobra command for cleaning evicted pods.
// It creates and returns a command with the given Use and Short descriptions,
// and sets the Run function to execute the cleanEvictedPods function.
func NewCommand() *cobra.Command {
	opts := &options.Options{}
	cmd := &cobra.Command{
		Use:   "clean-evicted",
		Short: "Clean evicted pods",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := k8sclient.NewClientset(k8sclient.FromContext(ctx))
			if err != nil {
				logger.FromContext(ctx).Error(err, "failed to create client")
				return err
			}
			return cleanEvictedPods(cmd.Context(), client, opts.Namespace())
		},
	}
	opts.BindCommonFlags(cmd)
	return cmd
}

// cleanEvictedPods cleans up evicted pods in the specified namespace.
func cleanEvictedPods(ctx context.Context, client kubernetes.Interface, namespace string) error {
	log := logger.FromContext(ctx)

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list pods")
		return err
	}

	var evictedPods []v1.Pod
	for _, pod := range pods.Items {
		if k8score.IsEvicted(nil, pod) {
			evictedPods = append(evictedPods, pod)
		}
	}

	deleted := 0
	for _, pod := range evictedPods {
		if err := k8score.DeletePod(ctx, client, pod); err != nil {
			log.Error(err, "failed to delete pod", "pod", fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		} else {
			deleted++
		}
	}

	log.Info("pods delete result", "deleted", deleted, "evicted", len(evictedPods))
	return nil
}
