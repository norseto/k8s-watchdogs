/*
MIT License

Copyright (c) 2019-2025 Norihiro Seto

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

package restartsts

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

// NewCommand returns a new Cobra command for restarting statefulsets.
func NewCommand() *cobra.Command {
	opts := &options.Options{}
	var allStatefulSets bool

	cmd := &cobra.Command{
		Use:   "restart-sts [statefulset-name|--all]",
		Short: "Restart statefulsets by name or all with --all",
		Long:  "Restart one or more statefulsets by specifying statefulset-name(s), or use --all to restart all in the namespace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			clnt, err := client.NewClientset(client.FromContext(ctx))
			if err != nil {
				logger.FromContext(ctx).Error(err, "failed to create client")
				return err
			}

			if allStatefulSets {
				return restartAllStatefulSets(ctx, clnt, opts.Namespace())
			}

			if len(args) < 1 {
				_ = cmd.Usage()
				return nil
			}
			return restartStatefulSet(ctx, clnt, opts.Namespace(), args)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if allStatefulSets {
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("requires at least one statefulset name or --all flag")
			}
			return nil
		},
	}
	opts.BindCommonFlags(cmd)
	cmd.Flags().BoolVarP(&allStatefulSets, "all", "a", false, "Restart all statefulsets in the namespace")

	return cmd
}

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;update

// restartAllStatefulSets restarts all statefulsets in the specified namespace
func restartAllStatefulSets(ctx context.Context, client kubernetes.Interface, namespace string) error {
	log := logger.FromContext(ctx)
	log.Info("Restarting all statefulsets", "namespace", namespace)

	// List all statefulsets in the namespace
	statefulsets, err := client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list statefulsets", "namespace", namespace)
		return err
	}

	if len(statefulsets.Items) == 0 {
		log.Info("No statefulsets found", "namespace", namespace)
		return nil
	}

	log.Info("Found statefulsets to restart", "count", len(statefulsets.Items))

	// Restart each statefulset
	var errorStatefulSets []string
	for _, sts := range statefulsets.Items {
		statefulsetCopy := sts.DeepCopy()
		err = kube.RestartStatefulSet(ctx, client, statefulsetCopy)
		if err != nil {
			log.Error(err, "failed to restart statefulset", "statefulset", sts.Name)
			errorStatefulSets = append(errorStatefulSets, sts.Name)
			continue
		}
		log.Info("Restarted statefulset", "statefulset", sts.Name)
	}

	if len(errorStatefulSets) > 0 {
		return fmt.Errorf("failed to restart statefulsets: %v", errorStatefulSets)
	}

	log.Info("Successfully restarted all statefulsets", "count", len(statefulsets.Items))
	return nil
}

func restartStatefulSet(ctx context.Context, client kubernetes.Interface, namespace string, targets []string) error {
	log := logger.FromContext(ctx)

	for _, target := range targets {
		sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, target, metav1.GetOptions{})
		if err != nil || sts == nil {
			log.Error(err, "failed to get statefulset", "target",
				fmt.Sprintf("%s/%s", namespace, target))
			return err
		}

		err = kube.RestartStatefulSet(ctx, client, sts)
		if err != nil {
			log.Error(err, "failed to restart statefulset", "target",
				fmt.Sprintf("%s/%s", namespace, target))
			return err
		}
		log.Info("Restarted statefulset", "statefulset", target)
	}
	return nil
}
