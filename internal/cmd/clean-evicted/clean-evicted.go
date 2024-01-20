package clean_evicted

import (
	"context"
	"github.com/norseto/k8s-watchdogs/pkg/k8sclient"
	"github.com/norseto/k8s-watchdogs/pkg/k8score"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// New returns a new Cobra command for cleaning evicted pods.
// It creates and returns a command with the given Use and Short descriptions,
// and sets the Run function to execute the cleanEvictedPods function.
func New() *cobra.Command {
	return &cobra.Command{
		Use:   "clean-evicted",
		Short: "Clean evicted pods",
		Run: func(cmd *cobra.Command, args []string) {
			cleanEvictedPods(cmd.Context())
		},
	}
}

func cleanEvictedPods(ctx context.Context) {
	var client kubernetes.Interface
	log := logger.FromContext(ctx)

	client, err := k8sclient.NewClientset()
	if err != nil {
		log.Error(err, "failed to create client")
		panic(err)
	}

	pods, err := client.CoreV1().Pods(v1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list pods")
		panic(err)
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
			log.Error(err, "failed to delete pod", "pod", pod)
		} else {
			deleted = deleted + 1
		}
	}
	log.Info("pods delete result", "deleted", deleted, "evicted", len(evictedPods))
}
