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
	"regexp"

	"k8s.io/client-go/kubernetes"

	"github.com/norseto/k8s-watchdogs/internal/options"
	"github.com/norseto/k8s-watchdogs/pkg/kube"
	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/spf13/cobra"
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
			cmd.SilenceUsage = true

			ctx := cmd.Context()
			clnt, err := client.NewClientset(client.FromContext(ctx))
			if err != nil {
				logger.FromContext(ctx).Error(err, "failed to create clientset")
				return err
			}
			return cleanEvictedPods(cmd.Context(), clnt, opts.Namespace())
		},
	}
	opts.BindCommonFlags(cmd)
	return cmd
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get

// cleanEvictedPods cleans up evicted pods in the specified namespace.
func cleanEvictedPods(ctx context.Context, client kubernetes.Interface, namespace string) error {
	log := logger.FromContext(ctx)

	// Security: Validate namespace parameter
	if err := validateNamespace(namespace); err != nil {
		log.Error(err, "invalid namespace parameter")
		return fmt.Errorf("invalid namespace: %w", err)
	}

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list pods", "namespace", namespace)
		return fmt.Errorf("failed to list pods: %w", err)
	}

	evictedPods := kube.FilterPods(pods, kube.IsEvictedPod)

	// Security: Limit the number of pods that can be deleted in one operation
	const maxDeletionsPerRun = 100
	if len(evictedPods) > maxDeletionsPerRun {
		log.Info("limiting pod deletions for safety", "found", len(evictedPods), "limit", maxDeletionsPerRun)
		evictedPods = evictedPods[:maxDeletionsPerRun]
	}

	deleted := 0
	var errs []error
	for _, pod := range evictedPods {
		// Security: Additional validation before deletion
		if pod == nil || pod.Name == "" {
			log.V(1).Info("skipping invalid pod")
			continue
		}

		if err := kube.DeletePod(ctx, client, *pod); err != nil {
			errs = append(errs, err)
			log.Error(err, "failed to delete pod", "pod", fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		} else {
			deleted++
			log.V(1).Info("deleted evicted pod", "pod", fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}
	}

	log.Info("pods delete result", "deleted", deleted, "evicted", len(evictedPods))
	if len(errs) > 0 {
		return fmt.Errorf("failed to delete %d pods", len(errs))
	}
	return nil
}

// validateNamespace validates the namespace parameter for security
func validateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	// Kubernetes namespace naming rules: lowercase alphanumeric and hyphens, max 63 chars
	validNamespace := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validNamespace.MatchString(namespace) || len(namespace) > 63 {
		return fmt.Errorf("invalid namespace format")
	}

	return nil
}
