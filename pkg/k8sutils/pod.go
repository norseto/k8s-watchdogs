package k8sutils

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IsPodReadyRunning checks if a given pod is in the "Running" phase and all its containers are ready.
func IsPodReadyRunning(_ context.Context, po v1.Pod) bool {
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

// DeletePod deletes a pod using the Kubernetes client.
func DeletePod(c kubernetes.Interface, pod v1.Pod) error {
	if err := c.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete Pod: "+pod.Name)
	}
	return nil
}

// toleratesTaint checks that the pod tolerated with a specific taint.
func toleratesTaint(pod *v1.Pod, taint v1.Taint) bool {
	for _, toleration := range pod.Spec.Tolerations {
		if toleration.ToleratesTaint(&taint) {
			return true
		}
	}
	return false
}
