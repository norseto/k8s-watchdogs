package main
// Evicted Pod Cleaner
// Deletes all evicted pod.

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/api/core/v1"
	common "github.com/norseto/k8s-watchdogs/pkg"
)

const (
	reasonEvicted = "Evicted"
)

func main() {
	var clientset *kubernetes.Clientset

	clientset, err := common.NewClientset()
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
		if err := deletePod(clientset, pod); err != nil {
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

// deletePod delete the pod
func deletePod(c *kubernetes.Clientset, pod v1.Pod) error {
	if err := c.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete Pod: "+pod.Name)
	}
	return nil
}
