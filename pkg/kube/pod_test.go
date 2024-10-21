/*
MIT License

Copyright (c) 2024 Norihiro Seto

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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestIsPodReadyRunning(t *testing.T) {
	tests := []struct {
		description string
		pod         corev1.Pod
		expected    bool
	}{
		{"Pod ready and running", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: true}}}}, true},
		{"Pod not running", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}, false},
		{"Pod running but not ready", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: false}}}}, false},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if IsPodReadyRunning(test.pod) != test.expected {
				t.Errorf("unexpected result for test %v", test.description)
			}
		})
	}
}

// TestDeletePod tests the DeletePod function
func TestDeletePod(t *testing.T) {
	ctx := context.TODO()

	// Define the namespace and name for our test pod.
	ns := "default"
	podName := "my-pod"

	// Create a fake client to mock API calls with.
	client := testclient.NewSimpleClientset()

	// Create a test pod.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      podName,
		},
	}

	_, err := client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("Error creating the test pod: %s", err)
	}

	// We call our function DeletePod using our fake client, a new context, and our test pod.
	err = DeletePod(ctx, client, *pod)
	assert.Nil(t, err)

	// We then check if the pod was deleted.
	// If DeletePod works as expected, it should have deleted the pod, so this API call should return an error.
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(pods.Items))
}

func TestToleratesTaint(t *testing.T) {
	myTaint := corev1.Taint{
		Key:   "myTaint",
		Value: "myValue",
	}

	tests := []struct {
		description string
		podSpec     *corev1.PodSpec
		expected    bool
	}{
		{"No tolerance for taint", &corev1.PodSpec{}, false},
		{"Toleration", &corev1.PodSpec{Tolerations: []corev1.Toleration{{Key: "myTaint", Operator: "Equal", Value: "myValue", Effect: ""}}}, true},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if toleratesTaint(test.podSpec, myTaint) != test.expected {
				t.Errorf("unexpected result for test %v", test.description)
			}
		})
	}
}

func TestIsEvicted(t *testing.T) {
	tests := []struct {
		name     string
		pod      corev1.Pod
		expected bool
	}{
		{
			name: "PodNotFailed",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			expected: false,
		},
		{
			name: "PodFailedButNotEvicted",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase:  corev1.PodFailed,
					Reason: "NotEvicted",
				},
			},
			expected: false,
		},
		{
			name: "PodFailedAndEvicted",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase:  corev1.PodFailed,
					Reason: reasonEvicted,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEvictedPod(&tt.pod)
			if got != tt.expected {
				t.Errorf("IsEvicted() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestGetPodRequestResources(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(2000, resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity(4096, resource.DecimalSI),
						},
					},
				},
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(1000, resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity(2048, resource.DecimalSI),
						},
					},
				},
			},
		},
	}

	expected := corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewQuantity(2000, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(4096, resource.DecimalSI),
	}

	result := GetPodRequestResources(pod.Spec)

	if result.Cpu().Cmp(*expected.Cpu()) != 0 {
		t.Errorf("CPU resource mismatch, expected: %v, got: %v", expected.Cpu(), result.Cpu())
	}

	if result.Memory().Cmp(*expected.Memory()) != 0 {
		t.Errorf("Memory resource mismatch, expected: %v, got: %v", expected.Memory(), result.Memory())
	}
}
