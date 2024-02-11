/*
MIT License

Copyright (c) 2019 Norihiro Seto

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

package kube

import (
	"context"
	"fmt"
	"github.com/norseto/k8s-watchdogs/pkg/generics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	reasonEvicted = "Evicted"
)

// IsPodReadyRunning checks if a given Pod is both ready and running.
func IsPodReadyRunning(po corev1.Pod) bool {
	phase := po.Status.Phase
	if phase != corev1.PodRunning && phase != "" {
		return false
	}
	for _, c := range po.Status.ContainerStatuses {
		if !c.Ready {
			return false
		}
	}
	return true
}

// GetPodRequestResources calculates the maximum CPU and memory resources requested by the containers in a given PodSpec.
// It iterates over each container in the PodSpec and checks if it has requested resources.
// If so, it compares the requested CPU and memory
// with the previously calculated maximums, and updates them if necessary.
// Finally, it returns the maximum CPU and memory resources as a corev1.ResourceList.
func GetPodRequestResources(podSpec corev1.PodSpec) corev1.ResourceList {
	maxCpu := *resource.NewQuantity(0, resource.DecimalSI)
	maxMem := *resource.NewQuantity(0, resource.DecimalSI)
	for _, c := range podSpec.Containers {
		if c.Resources.Requests == nil {
			continue
		}
		if c.Resources.Requests.Cpu().Cmp(maxCpu) > 0 {
			maxCpu = c.Resources.Requests.Cpu().DeepCopy()
		}
		if c.Resources.Requests.Memory().Cmp(maxMem) > 0 {
			maxMem = c.Resources.Requests.Memory().DeepCopy()
		}
	}

	ret := corev1.ResourceList{
		corev1.ResourceCPU:    maxCpu,
		corev1.ResourceMemory: maxMem,
	}
	return ret
}

// DeletePod deletes a pod using the Kubernetes client.
func DeletePod(ctx context.Context, client kubernetes.Interface, pod corev1.Pod) error {
	if err := client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Pod: %s, %w", pod.Name, err)
	}
	return nil
}

// toleratesTaint checks if a given PodSpec tolerates a specific Taint.
func toleratesTaint(podSpec *corev1.PodSpec, taint corev1.Taint) bool {
	for _, toleration := range podSpec.Tolerations {
		if toleration.ToleratesTaint(&taint) {
			return true
		}
	}
	return false
}

// IsEvicted returns the pod is already Evicted.
// Deprecated: Use IsEvictedPod instead
func IsEvicted(_ context.Context, pod corev1.Pod) bool {
	return IsEvictedPod(&pod)
}

// IsEvictedPod checks if a given Pod has been evicted.
// It returns true if the Pod's phase is "Failed" and the reason is "Evicted",
// otherwise, it returns false.
func IsEvictedPod(pod *corev1.Pod) bool {
	status := pod.Status
	if status.Phase == corev1.PodFailed && status.Reason == reasonEvicted {
		return true
	}
	return false
}

// FilterPods filters the given list of Pods using the provided filter function and returns a list of filtered Pods.
func FilterPods(list *corev1.PodList, filter func(*corev1.Pod) bool) []*corev1.Pod {
	var filtered []*corev1.Pod
	generics.Each(list.Items, func(item corev1.Pod) {
		if filter(&item) {
			filtered = append(filtered, &item)
		}
	})
	return filtered
}
