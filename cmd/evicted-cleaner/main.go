package main

// Evicted Pod Cleaner
// Deletes all evicted pod.

import (
	"context"
	"github.com/norseto/k8s-watchdogs/pkg/k8sclient"
	"github.com/norseto/k8s-watchdogs/pkg/k8score"
	"github.com/norseto/k8s-watchdogs/pkg/log"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func main() {
	var client kubernetes.Interface

	ctx := log.WithContext(context.Background(), log.InitLogger())
	logger := log.FromContext(ctx, "cmd", "evicted-cleaner")

	client, err := k8sclient.NewClientset()
	if err != nil {
		logger.Error(err, "failed to create client")
		panic(err)
	}

	pods, err := client.CoreV1().Pods(v1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to list pods")
		panic(err)
	}

	var evicteds []v1.Pod
	for _, pod := range pods.Items {
		if k8score.IsEvicted(nil, pod) {
			evicteds = append(evicteds, pod)
		}
	}

	deleted := 0
	for _, pod := range evicteds {
		if err := k8score.DeletePod(ctx, client, pod); err != nil {
			logger.Error(err, "failed to delete pod", "pod", pod)
		} else {
			deleted = deleted + 1
		}
	}
	logger.Info("pods delete result", "deleted", deleted, "evicted", len(evicteds))
}
