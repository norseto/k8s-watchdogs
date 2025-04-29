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
	var allDeployments bool

	cmd := &cobra.Command{
		Use:   "restart-deploy [deployment-name|--all]",
		Short: "Restart deployments by name or all with --all",
		Long:  "Restart one or more deployments by specifying deployment-name(s), or use --all to restart all in the namespace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			clnt, err := client.NewClientset(client.FromContext(ctx))
			if err != nil {
				logger.FromContext(ctx).Error(err, "failed to create client")
				return err
			}

			if allDeployments {
				return restartAllDeployments(ctx, clnt, opts.Namespace())
			}

			if len(args) < 1 {
				_ = cmd.Usage()
				return nil
			}
			return restartDeployment(ctx, clnt, opts.Namespace(), args)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if allDeployments {
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("requires at least one deployment name or --all flag")
			}
			return nil
		},
	}
	opts.BindCommonFlags(cmd)
	cmd.Flags().BoolVarP(&allDeployments, "all", "a", false, "Restart all deployments in the namespace")

	return cmd
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;update

// restartAllDeployments restarts all deployments in the specified namespace
func restartAllDeployments(ctx context.Context, client kubernetes.Interface, namespace string) error {
	log := logger.FromContext(ctx)
	log.Info("Restarting all deployments", "namespace", namespace)

	// List all deployments in the namespace
	deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list deployments", "namespace", namespace)
		return err
	}

	if len(deployments.Items) == 0 {
		log.Info("No deployments found", "namespace", namespace)
		return nil
	}

	log.Info("Found deployments to restart", "count", len(deployments.Items))

	// Restart each deployment
	errors := 0
	for _, dep := range deployments.Items {
		deploymentCopy := dep.DeepCopy()
		err = kube.RestartDeployment(ctx, client, deploymentCopy)
		if err != nil {
			log.Error(err, "failed to restart deployment", "deployment", dep.Name)
			errors++
			// Continue to restart other deployments even if one fails
			continue
		}
		log.Info("Restarted deployment", "deployment", dep.Name)
	}

	if errors > 0 {
		return fmt.Errorf("failed to restart %d deployments", errors)
	}

	log.Info("Successfully restarted all deployments", "count", len(deployments.Items))
	return nil
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
		log.Info("Restarted deployment", "deployment", target)
	}
	return nil
}
