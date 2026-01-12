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

package rebalancer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
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

func TestSpecReplicasNilStates(t *testing.T) {
	tests := []struct {
		name string
		reb  *Rebalancer
	}{
		{name: "nil current", reb: &Rebalancer{}},
		{name: "nil replicaset", reb: &Rebalancer{current: &ReplicaState{}}},
		{name: "nil spec replicas", reb: &Rebalancer{current: &ReplicaState{Replicaset: &appsv1.ReplicaSet{}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.reb.specReplicas(); got != 0 {
				t.Fatalf("expected 0 replicas, got %d", got)
			}
		})
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

func TestCurrentReplicasNilStates(t *testing.T) {
	tests := []struct {
		name string
		reb  *Rebalancer
	}{
		{name: "nil current", reb: &Rebalancer{}},
		{name: "nil replicaset", reb: &Rebalancer{current: &ReplicaState{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.reb.currentReplicas(); got != 0 {
				t.Fatalf("expected 0 replicas, got %d", got)
			}
		})
	}
}

func node(name string, opt ...func(n *corev1.Node)) *corev1.Node {
	n := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
	for _, o := range opt {
		o(n)
	}
	return n
}

func pod(name, node string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       corev1.PodSpec{NodeName: node},
	}
}

func capacity(cpu, memory string) func(n *corev1.Node) {
	return func(n *corev1.Node) {
		res := corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(memory),
		}
		n.Status.Capacity = res
		n.Status.Allocatable = res.DeepCopy()
	}
}

func TestFilterSchedulables(t *testing.T) {
	// Create a test ReplicaState
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
	}
	podStatus := &PodStatus{
		Pod: pod,
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
	rebalancer.filterSchedulables(context.TODO())

	// Check the filtered Nodes
	expected := []*corev1.Node{node("node-1")}
	actual := rebalancer.current.Nodes
	if len(actual) != len(expected) {
		t.Errorf("Expected %d Nodes, but got %d", len(expected), len(actual))
	} else {
		for i := range actual {
			if actual[i].Name != expected[i].Name {
				t.Errorf("Expected Node %s, but got %s", expected[i].Name, actual[i].Name)
			}
		}
	}
}

func TestFilterSchedulablesEdgeCases(t *testing.T) {
	t.Run("no pod status entries", func(t *testing.T) {
		reb := &Rebalancer{current: &ReplicaState{Nodes: []*corev1.Node{node("keep")}}}
		original := append([]*corev1.Node{}, reb.current.Nodes...)
		reb.filterSchedulables(context.Background())
		if len(reb.current.Nodes) != len(original) {
			t.Fatalf("expected nodes unchanged")
		}
	})

	t.Run("first pod nil", func(t *testing.T) {
		reb := &Rebalancer{current: &ReplicaState{PodStatus: []*PodStatus{{Pod: nil}}, Nodes: []*corev1.Node{node("keep")}}}
		reb.filterSchedulables(context.Background())
		if len(reb.current.Nodes) != 1 {
			t.Fatalf("expected nodes unchanged when first pod nil")
		}
	})
}

func TestRebalance(t *testing.T) {
	replicas := int32(3)
	ctx := context.Background()

	// Create a test ReplicaState
	replicaSet := &appsv1.ReplicaSet{
		Status: appsv1.ReplicaSetStatus{
			Replicas: replicas,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
		},
	}
	node1, node2, node3 :=
		node("node-1", capacity("100m", "100Mi")),
		node("node-2", capacity("200m", "200Mi")),
		node("node-3", capacity("100m", "100Mi"))
	pod1, pod2, pod3 :=
		pod("pod-1", "node-1"),
		pod("pod-2", "node-2"),
		pod("pod-3", "node-2")

	replicaState := &ReplicaState{
		Replicaset: replicaSet,
		Nodes:      []*corev1.Node{node1, node2, node3},
		PodStatus: []*PodStatus{
			{Pod: pod1},
			{Pod: pod2},
			{Pod: pod3},
		},
	}
	rebalancer := NewRebalancer(ctx, replicaState, .25)

	// Create a mock kubernetes.Interface
	mockClient := fake.NewSimpleClientset(replicaSet, node1, node2, node3, pod1, pod2, pod3)

	// Call the Rebalance function
	_, err := rebalancer.Rebalance(ctx, mockClient)

	// Check for any errors
	if err != nil {
		t.Errorf("Rebalance returned an error: %v", err)
	}

	// Check if the pod on the node with the most pods was deleted.
	deletedPodNode := ""
	for _, p := range rebalancer.current.PodStatus {
		if p.deleted {
			deletedPodNode = p.Pod.Spec.NodeName
			break
		}
	}
	assert.Equal(t, "node-2", deletedPodNode, "Pod on node-2 should be deleted")
}

func TestRebalanceEarlyExitConditions(t *testing.T) {
	replicas := int32(1)
	ctx := context.Background()
	baseState := &ReplicaState{PodStatus: []*PodStatus{{Pod: pod("p", "n")}}}

	cases := []struct {
		name    string
		current *ReplicaState
		rate    float32
	}{
		{
			name: "single node", rate: .5,
			current: &ReplicaState{
				Replicaset: &appsv1.ReplicaSet{Spec: appsv1.ReplicaSetSpec{Replicas: int32Ptr(2)}, Status: appsv1.ReplicaSetStatus{Replicas: 2}},
				Nodes:      []*corev1.Node{node("only")},
				PodStatus:  []*PodStatus{{Pod: pod("p", "only")}},
			},
		},
		{
			name: "insufficient spec replicas", rate: .5,
			current: &ReplicaState{
				Replicaset: &appsv1.ReplicaSet{Spec: appsv1.ReplicaSetSpec{Replicas: int32Ptr(1)}, Status: appsv1.ReplicaSetStatus{Replicas: 1}},
				Nodes:      []*corev1.Node{node("n1"), node("n2")},
				PodStatus:  baseState.PodStatus,
			},
		},
		{
			name: "current replicas below spec", rate: .5,
			current: &ReplicaState{
				Replicaset: &appsv1.ReplicaSet{Spec: appsv1.ReplicaSetSpec{Replicas: int32Ptr(3)}, Status: appsv1.ReplicaSetStatus{Replicas: 2}},
				Nodes:      []*corev1.Node{node("n1"), node("n2")},
				PodStatus:  []*PodStatus{{Pod: pod("p1", "n1")}, {Pod: pod("p2", "n2")}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reb := NewRebalancer(ctx, tc.current, tc.rate)
			acted, err := reb.Rebalance(ctx, fake.NewSimpleClientset())
			assert.NoError(t, err)
			assert.False(t, acted)
		})
	}

	// ensure different rate with replicas < 2 early exit branch.
	reb := NewRebalancer(ctx, &ReplicaState{Replicaset: &appsv1.ReplicaSet{Spec: appsv1.ReplicaSetSpec{Replicas: &replicas}, Status: appsv1.ReplicaSetStatus{Replicas: replicas}}, Nodes: []*corev1.Node{node("n1"), node("n2")}, PodStatus: baseState.PodStatus}, .2)
	acted, err := reb.Rebalance(ctx, fake.NewSimpleClientset())
	assert.NoError(t, err)
	assert.False(t, acted)
}

func TestRebalanceHandlesDeleteErrors(t *testing.T) {
	replicas := int32(4)
	ctx := context.Background()
	rs := &appsv1.ReplicaSet{
		Spec:   appsv1.ReplicaSetSpec{Replicas: &replicas},
		Status: appsv1.ReplicaSetStatus{Replicas: replicas},
	}
	pods := []*PodStatus{
		{Pod: pod("p1", "n1")},
		{Pod: pod("p2", "n1")},
		{Pod: pod("p3", "n1")},
		{Pod: pod("p4", "n2")},
	}
	nodes := []*corev1.Node{node("n1", capacity("100m", "100Mi")), node("n2", capacity("100m", "100Mi"))}
	reb := NewRebalancer(ctx, &ReplicaState{Replicaset: rs, Nodes: nodes, PodStatus: pods}, .5)

	client := fake.NewSimpleClientset(rs, nodes[0], nodes[1], pods[0].Pod, pods[1].Pod, pods[2].Pod, pods[3].Pod)
	delErr := errors.New("delete failed")
	client.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, delErr
	})

	acted, err := reb.Rebalance(ctx, client)
	if err == nil || !strings.Contains(err.Error(), "failed to delete Pod") {
		t.Fatalf("expected wrapped delete error, got %v", err)
	}
	if acted {
		t.Fatalf("expected acted to be false when delete fails")
	}
}

func TestRebalanceReturnsWhenNoPodsToDelete(t *testing.T) {
	replicas := int32(2)
	ctx := context.Background()
	rs := &appsv1.ReplicaSet{Spec: appsv1.ReplicaSetSpec{Replicas: &replicas}, Status: appsv1.ReplicaSetStatus{Replicas: replicas}}
	pods := []*PodStatus{{Pod: pod("p1", "n1"), deleted: true}, {Pod: pod("p2", "n2"), deleted: true}}
	nodes := []*corev1.Node{node("n1", capacity("100m", "100Mi")), node("n2", capacity("100m", "100Mi"))}
	reb := NewRebalancer(ctx, &ReplicaState{Replicaset: rs, Nodes: nodes, PodStatus: pods}, .5)

	acted, err := reb.Rebalance(ctx, fake.NewSimpleClientset())
	assert.NoError(t, err)
	assert.False(t, acted)
}

func TestDeletePodOnNodeNoMatch(t *testing.T) {
	r := &Rebalancer{current: &ReplicaState{PodStatus: []*PodStatus{{Pod: pod("p1", "n1"), deleted: true}}, Nodes: []*corev1.Node{node("n1")}}}
	err := r.deletePodOnNode(context.Background(), fake.NewSimpleClientset(), "n1")
	if err != nil {
		t.Fatalf("expected nil error when nothing deleted, got %v", err)
	}
}

func TestDeletePodOnNode(t *testing.T) {
	// Create a test ReplicaState
	replicaState := &ReplicaState{
		PodStatus: []*PodStatus{
			{Pod: pod("pod-1", "node-1")},
			{Pod: pod("pod-2", "node-2")},
		},
		Nodes: []*corev1.Node{
			node("node-1"),
			node("node-2"),
			node("node-3"),
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
	err := rebalancer.deletePodOnNode(ctx, client, "node-1")

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
			{Pod: pod("pod-1", "node-1")},
			{Pod: pod("pod-2", "node-2")},
			{Pod: pod("pod-3", "node-2")},
			nil,
			{Pod: pod("pod-4", "node-3")},
			{Pod: pod("pod-5", "node-3")},
			{Pod: pod("pod-6", "node-3")},
		},
		Nodes: []*corev1.Node{
			node("node-1"),
			nil,
			node("node-2"),
			node("node-3"),
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
			{Pod: pod("pod-1", "node-1")},
			{Pod: pod("pod-2", "node-2")},
			{Pod: pod("pod-3", "node-2")},
			{Pod: pod("pod-4", "node-3")},
			{Pod: pod("pod-5", "node-3")},
			{Pod: pod("pod-6", "node-3")},
		},
		Nodes: []*corev1.Node{
			node("node-1"),
			node("node-2"),
			node("node-3"),
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
