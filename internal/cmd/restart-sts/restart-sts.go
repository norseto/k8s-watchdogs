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
	"regexp"

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

			// Security: Validate namespace parameter
			if err := validateNamespace(opts.Namespace()); err != nil {
				logger.FromContext(ctx).Error(err, "invalid namespace parameter")
				return fmt.Errorf("invalid namespace: %w", err)
			}

			// Security: Validate statefulset names
			for _, name := range args {
				if err := validateResourceName(name); err != nil {
					logger.FromContext(ctx).Error(err, "invalid statefulset name", "name", name)
					return fmt.Errorf("invalid statefulset name %s: %w", name, err)
				}
			}

			cmd.SilenceUsage = true

			clnt, err := client.NewClientset(client.FromContext(ctx))
			if err != nil {
				logger.FromContext(ctx).Error(err, "failed to create client")
				return fmt.Errorf("failed to create client: %w", err)
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

	// Security: Limit the number of statefulsets that can be restarted in one operation
	const maxRestartsPerRun = 50
	if len(targets) > maxRestartsPerRun {
		return fmt.Errorf("too many statefulsets specified: %d (max: %d)", len(targets), maxRestartsPerRun)
	}

	for _, target := range targets {
		sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, target, metav1.GetOptions{})
		if err != nil {
			log.Error(err, "failed to get statefulset", "target", fmt.Sprintf("%s/%s", namespace, target))
			return fmt.Errorf("failed to get statefulset %s/%s: %w", namespace, target, err)
		}

		if sts == nil {
			return fmt.Errorf("statefulset %s/%s not found", namespace, target)
		}

		err = kube.RestartStatefulSet(ctx, client, sts)
		if err != nil {
			log.Error(err, "failed to restart statefulset", "target", fmt.Sprintf("%s/%s", namespace, target))
			return fmt.Errorf("failed to restart statefulset %s/%s: %w", namespace, target, err)
		}
		log.Info("Restarted statefulset", "statefulset", target, "namespace", namespace)
	}
	return nil
}

// validateResourceName validates Kubernetes resource names for security
func validateResourceName(name string) error {
	if name == "" {
		return fmt.Errorf("resource name cannot be empty")
	}

	if len(name) > 253 {
		return fmt.Errorf("resource name too long: %d characters", len(name))
	}

	// Kubernetes resource naming rules: lowercase alphanumeric, hyphens, and dots
	validName := regexp.MustCompile(`^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid resource name format")
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
