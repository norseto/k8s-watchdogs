package k8sutils

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"
)

func TestIsPodReadyRunning(t *testing.T) {
	// Positive test case
	pod := v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{Ready: true},
				{Ready: true},
			},
		},
	}
	result := IsPodReadyRunning(context.TODO(), pod)
	if !result {
		t.Errorf("Expected true, but got false")
	}

	// Negative test case
	pod = v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodPending,
			ContainerStatuses: []v1.ContainerStatus{
				{Ready: true},
				{Ready: false},
			},
		},
	}
	result = IsPodReadyRunning(context.TODO(), pod)
	if result {
		t.Errorf("Expected false, but got true")
	}
}

func TestDeletePod(t *testing.T) {
	// Positive test case
	ctx := context.TODO()
	client := fake.NewSimpleClientset(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	})
	err := DeletePod(ctx, client, v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Negative test case
	ctx = context.TODO()
	client = fake.NewSimpleClientset()
	err = DeletePod(ctx, client, v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "non-existent-pod",
			Namespace: "default",
		},
	})
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
}
