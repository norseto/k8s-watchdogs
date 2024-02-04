package main

// Evicted Pod Cleaner
// Deletes all evicted pod.

import (
	"context"
	"flag"
	"github.com/norseto/k8s-watchdogs/pkg/kube"
	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func main() {
	var kubeClient kubernetes.Interface

	ctx := logger.WithContext(context.Background(), logger.InitLogger())
	log := logger.FromContext(ctx, "cmd", "evicted-cleaner")

	opt := &client.Options{}
	opt.BindFlags(flag.CommandLine)
	flag.Parse()

	kubeClient, err := client.NewClientset(opt)
	if err != nil {
		log.Error(err, "failed to create client")
		panic(err)
	}

	pods, err := kubeClient.CoreV1().Pods(v1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list pods")
		panic(err)
	}

	var evicteds []v1.Pod
	for _, pod := range pods.Items {
		if kube.IsEvicted(nil, pod) {
			evicteds = append(evicteds, pod)
		}
	}

	deleted := 0
	for _, pod := range evicteds {
		if err := kube.DeletePod(ctx, kubeClient, pod); err != nil {
			log.Error(err, "failed to delete pod", "pod", pod)
		} else {
			deleted = deleted + 1
		}
	}
	log.Info("pods delete result", "deleted", deleted, "evicted", len(evicteds))
}
