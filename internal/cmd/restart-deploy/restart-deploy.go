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
