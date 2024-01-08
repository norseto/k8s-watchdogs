package k8sutils

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
)

// MockKubernetesClient is a mock implementation of the kubernetes.Interface interface
type MockKubernetesClient struct {
	mock.Mock
}

func (m *MockKubernetesClient) CoreV1() *MockCoreV1Interface {
	ret := m.Called()
	var r0 *MockCoreV1Interface
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*MockCoreV1Interface)
	}
	return r0
}

type MockCoreV1Interface struct {
	mock.Mock
}

func (m *MockCoreV1Interface) Pods(namespace string) *MockPodInterface {
	ret := m.Called(namespace)
	var r0 *MockPodInterface
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*MockPodInterface)
	}
	return r0
}

type MockPodInterface struct {
	mock.Mock
}

func (m *MockPodInterface) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	ret := m.Called(ctx, name, options)
	var r0 error
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(error)
	}
	return r0
}

func TestIsPodReadyRunning(t *testing.T) {
	tests := []struct {
		description string
		pod         v1.Pod
		expected    bool
	}{
		{"Pod ready and running", v1.Pod{Status: v1.PodStatus{Phase: v1.PodRunning, ContainerStatuses: []v1.ContainerStatus{{Ready: true}}}}, true},
		{"Pod not running", v1.Pod{Status: v1.PodStatus{Phase: v1.PodPending}}, false},
		{"Pod running but not ready", v1.Pod{Status: v1.PodStatus{Phase: v1.PodRunning, ContainerStatuses: []v1.ContainerStatus{{Ready: false}}}}, false},
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
	pod := &v1.Pod{
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
	myTaint := v1.Taint{
		Key:   "myTaint",
		Value: "myValue",
	}

	tests := []struct {
		description string
		podSpec     *v1.PodSpec
		expected    bool
	}{
		{"No tolerance for taint", &v1.PodSpec{}, false},
		{"Toleration", &v1.PodSpec{Tolerations: []v1.Toleration{{Key: "myTaint", Operator: "Equal", Value: "myValue", Effect: ""}}}, true},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if toleratesTaint(test.podSpec, myTaint) != test.expected {
				t.Errorf("unexpected result for test %v", test.description)
			}
		})
	}
}
