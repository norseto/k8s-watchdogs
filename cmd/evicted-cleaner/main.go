package main

// Evicted Pod Cleaner
// Deletes all evicted pod.

import (
	k8sutils "github.com/norseto/k8s-watchdogs/pkg/k8sutils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	reasonEvicted = "Evicted"
)

func main() {
	var clientset *kubernetes.Clientset

	clientset, err := k8sutils.NewClientset()
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to create clientset"))
	}

	pods, err := clientset.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list pods"))
	}

	evicteds := [](v1.Pod){}
	for _, pod := range pods.Items {
		if isEvicted(pod) {
			evicteds = append(evicteds, pod)
		}
	}

	deleted := 0
	for _, pod := range evicteds {
		if err := k8sutils.DeletePod(clientset, pod); err != nil {
			log.Info(err)
		} else {
			deleted = deleted + 1
		}
	}
	log.Info("removed ", deleted, " pods (evicted: ", len(evicteds), ")")
}

// isEvicted returns the pod is already Evicted
func isEvicted(pod v1.Pod) bool {
	status := pod.Status
	if status.Phase == v1.PodFailed && status.Reason == reasonEvicted {
		return true
	}
	return false
}
