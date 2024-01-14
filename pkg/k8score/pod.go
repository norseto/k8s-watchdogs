package k8score

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	reasonEvicted = "Evicted"
)

// IsPodReadyRunning checks if a given Pod is both ready and running.
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

// DeletePod deletes a pod using the Kubernetes client.
func DeletePod(ctx context.Context, client kubernetes.Interface, pod v1.Pod) error {
	if err := client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Pod: %s, %w", pod.Name, err)
	}
	return nil
}

// toleratesTaint checks if a given PodSpec tolerates a specific Taint.
func toleratesTaint(podSpec *v1.PodSpec, taint v1.Taint) bool {
	for _, toleration := range podSpec.Tolerations {
		if toleration.ToleratesTaint(&taint) {
			return true
		}
	}
	return false
}

// IsEvicted returns the pod is already Evicted
func IsEvicted(_ context.Context, pod v1.Pod) bool {
	status := pod.Status
	if status.Phase == v1.PodFailed && status.Reason == reasonEvicted {
		return true
	}
	return false
}
