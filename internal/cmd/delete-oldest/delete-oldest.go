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

package deleteoldest

import (
	"context"
	"github.com/norseto/k8s-watchdogs/internal/options"
	"github.com/norseto/k8s-watchdogs/pkg/k8sclient"
	"github.com/norseto/k8s-watchdogs/pkg/k8score"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
)

// NewCommand returns a new Cobra command for re-balancing pods.
func NewCommand() *cobra.Command {
	var prefix string
	var minPods int

	opts := &options.Options{}
	cmd := &cobra.Command{
		Use:   "delete-oldest",
		Short: "Delete oldest pod(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if prefix == "" || minPods < 1 {
				_ = cmd.Usage()
				return nil
			}
			
			ctx := cmd.Context()
			client, err := k8sclient.NewClientset(k8sclient.FromContext(ctx))
			if err != nil {
				logger.FromContext(ctx).Error(err, "failed to create client")
				return err
			}
			return deleteOldestPods(cmd.Context(), client, opts.Namespace(), prefix, minPods)
		},
	}
	opts.BindCommonFlags(cmd)

	flg := cmd.Flags()
	flg.StringVarP(&prefix, "prefix", "p", "", "Pod name prefix to delete.")
	flg.IntVarP(&minPods, "minPods", "m", 3, "Min pods required.")

	return cmd
}

func deleteOldestPods(ctx context.Context, client kubernetes.Interface, namespace, prefix string, minPods int) error {

	log := logger.FromContext(ctx)

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list pods")
		return err
	}

	picked, err := pickOldest(prefix, minPods, pods.Items)
	if err != nil {
		log.Error(err, "failed to pick oldest pod")
		return err
	}
	if err := k8score.DeletePod(ctx, client, *picked); err != nil {
		log.Error(err, "failed to delete pod")
		return err
	}
	log.Info("removed", "pod",
		picked.ObjectMeta.Namespace+"/"+picked.ObjectMeta.Name)

	return nil
}

func pickOldest(prefix string, min int, pods []corev1.Pod) (*corev1.Pod, error) {
	var oldest corev1.Pod
	count := 0
	for _, p := range pods {
		if !k8score.IsPodReadyRunning(p) || !strings.HasPrefix(p.ObjectMeta.Name, prefix) {
			continue
		}
		if oldest.Status.StartTime == nil ||
			oldest.Status.StartTime.Time.After(p.Status.StartTime.Time) {
			oldest = p
		}
		count++
	}
	if count >= min {
		return &oldest, nil
	}
	return nil, errors.Errorf("Found only %v pods. Should at least %v pods running.", count, min)
}
