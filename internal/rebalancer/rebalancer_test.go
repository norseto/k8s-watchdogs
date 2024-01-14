package rebalancer

import (
	"context"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func TestSpecReplicas(t *testing.T) {
	// Create a test ReplicaState
	replicaSet := &appsv1.ReplicaSet{
		Spec: appsv1.ReplicaSetSpec{
			Replicas: int32Ptr(5),
		},
	}
	replicaState := &ReplicaState{
		Replicaset: replicaSet,
	}

	// Create a test Rebalancer
	rebalancer := &Rebalancer{
		current: replicaState,
	}

	// Call the specReplicas function
	actual := rebalancer.specReplicas()

	// Check the expected result
	expected := int32(5)
	if actual != expected {
		t.Errorf("Expected: %d, but got: %d", expected, actual)
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}

func TestCurrentReplicas(t *testing.T) {
	// Create a test ReplicaState
	replicaSet := &appsv1.ReplicaSet{
		Status: appsv1.ReplicaSetStatus{
			Replicas: 5,
		},
	}
	replicaState := &ReplicaState{
		Replicaset: replicaSet,
	}

	// Create a test Rebalancer
	rebalancer := &Rebalancer{
		current: replicaState,
	}

	// Call the currentReplicas function
	actual := rebalancer.currentReplicas()

	// Check the expected result
	expected := int32(5)
	if actual != expected {
		t.Errorf("Expected: %d, but got: %d", expected, actual)
	}
}

func node(name string, opt ...func(n *corev1.Node)) *corev1.Node {
	n := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
	for _, o := range opt {
		o(n)
	}
	return n
}

func TestFilterSchedulables(t *testing.T) {
	// Create a test ReplicaState
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
	}
	podStatus := &PodStatus{
		Pod:  pod,
		Node: node("node-1"),
	}
	replicaState := &ReplicaState{
		PodStatus: []*PodStatus{podStatus},
		Nodes: []*corev1.Node{
			node("node-1"),
			node("node-2", func(n *corev1.Node) {
				n.Spec.Unschedulable = true
			}),
		},
	}

	// Create a test Rebalancer
	rebalancer := &Rebalancer{
		current: replicaState,
	}

	// Call the filterSchedulables function
	rebalancer.filterSchedulables()

	// Check the filtered Nodes
	expected := []*corev1.Node{node("node-1")}
	actual := rebalancer.current.Nodes
	if len(actual) != len(expected) {
		t.Errorf("Expected %d Nodes, but got %d", len(expected), len(actual))
	} else {
		for i := 0; i < len(actual); i++ {
			if actual[i].Name != expected[i].Name {
				t.Errorf("Expected Node %s, but got %s", expected[i].Name, actual[i].Name)
			}
		}
	}
}

func TestRebalance(t *testing.T) {
	// Create a test ReplicaState
	replicaSet := &appsv1.ReplicaSet{
		Status: appsv1.ReplicaSetStatus{
			Replicas: 5,
		},
	}
	replicaState := &ReplicaState{
		Replicaset: replicaSet,
		Nodes: []*corev1.Node{
			node("node-1"),
			node("node-2"),
			node("node-3"),
		},
		PodStatus: []*PodStatus{
			{Pod: &corev1.Pod{Spec: corev1.PodSpec{NodeName: "node-1"}}},
			{Pod: &corev1.Pod{Spec: corev1.PodSpec{NodeName: "node-2"}}},
			{Pod: &corev1.Pod{Spec: corev1.PodSpec{NodeName: "node-3"}}},
		},
	}
	rebalancer := &Rebalancer{
		current:          replicaState,
		maxRebalanceRate: 0.25,
	}

	// Create a mock kubernetes.Interface
	mockClient := &kubernetes.Clientset{}

	// Call the Rebalance function
	_, err := rebalancer.Rebalance(context.Background(), mockClient)

	// Check for any errors
	if err != nil {
		t.Errorf("Rebalance returned an error: %v", err)
	}
}

func TestDeletePodOnNode(t *testing.T) {
	// Create a test ReplicaState
	replicaState := &ReplicaState{
		PodStatus: []*PodStatus{
			{Pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-1"}}, Node: node("node-1")},
			{Pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-2"}}, Node: node("node-2")},
		},
	}
	rebalancer := &Rebalancer{
		current: replicaState,
	}

	// Create a mock kubernetes.Interface
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	for _, ps := range replicaState.PodStatus {
		// Create a mock Pod
		_, err := client.CoreV1().Pods(ps.Pod.Namespace).Create(ctx, ps.Pod, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	// Call the deletePodOnNode function
	err := rebalancer.deletePodOnNode(context.Background(), client, "node-1")

	// Check for any errors
	if err != nil {
		t.Errorf("deletePodOnNode returned an error: %v", err)
	}

	// Check if the Pod was marked as deleted
	if !rebalancer.current.PodStatus[0].deleted {
		t.Errorf("PodStatus[0] was not marked as deleted")
	}
}

func TestGetNodeWithMaxPods(t *testing.T) {
	// Create a test ReplicaState
	replicaState := &ReplicaState{
		PodStatus: []*PodStatus{
			{Node: node("node-1")},
			{Node: node("node-2")},
			{Node: node("node-2")},
			{Node: node("node-3")},
			{Node: node("node-3")},
			{Node: node("node-3")},
		},
	}
	rebalancer := &Rebalancer{
		current: replicaState,
	}

	// Call the getNodeWithMaxPods function
	node, count := rebalancer.getNodeWithMaxPods()

	// Check the expected result
	expectedNode := "node-3"
	expectedCount := 3
	if node != expectedNode || count != expectedCount {
		t.Errorf("Expected: %s, %d, but got: %s, %d", expectedNode, expectedCount, node, count)
	}
}

func TestCountPodsPerNode(t *testing.T) {
	// Create a test ReplicaState
	replicaState := &ReplicaState{
		PodStatus: []*PodStatus{
			{Node: node("node-1")},
			{Node: node("node-2")},
			{Node: node("node-2")},
			{Node: node("node-3")},
			{Node: node("node-3")},
			{Node: node("node-3")},
		},
	}
	rebalancer := &Rebalancer{
		current: replicaState,
	}

	// Call the countPodsPerNode function
	podCounts := rebalancer.countPodsPerNode()

	// Check the expected result
	expectedCounts := map[string]int{
		"node-1": 1,
		"node-2": 2,
		"node-3": 3,
	}
	for nodeName, expectedCount := range expectedCounts {
		actualCount, ok := podCounts[nodeName]
		if !ok {
			t.Errorf("Expected pod count for Node %s, but got 0", nodeName)
		} else if actualCount != expectedCount {
			t.Errorf("Expected pod count for Node %s: %d, but got: %d", nodeName, expectedCount, actualCount)
		}
	}
}