package k8sutils

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IsPodReadyRunning returns the Pod phase is Running or empty and all container is ready.
// Parameter:
//   po v1.Pod : Target Pod
// Returns:
//   bool : True if pod is ready(Running and all containers are also ready)
func IsPodReadyRunning(po v1.Pod) bool {
	phase := po.Status.Phase
	if phase != v1.PodRunning && phase != "" {
		return false
	}
	for _, c := range po.Status.ContainerStatuses {
		if !c.Ready {
			return false
		}
	}
	return true
}

// DeletePod delete the pod
func DeletePod(c *kubernetes.Clientset, pod v1.Pod) error {
	if err := c.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete Pod: "+pod.Name)
	}
	return nil
}
