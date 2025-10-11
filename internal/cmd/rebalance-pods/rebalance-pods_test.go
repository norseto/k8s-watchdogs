/*
MIT License

Copyright (c) 2019-2024 Norihiro Seto

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

package rebalancepods

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/norseto/k8s-watchdogs/internal/pkg/validation"
	"github.com/norseto/k8s-watchdogs/internal/rebalancer"
	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func schedulableNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2000m"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	}
}

func readyPod(name, node string, owner metav1.OwnerReference) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		Spec: corev1.PodSpec{NodeName: node},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}
}

func int32Ptr(v int32) *int32 {
	return &v
}

type fakeRebalancer struct {
	result bool
	err    error
}

func (f fakeRebalancer) Rebalance(context.Context, kubernetes.Interface) (bool, error) {
	return f.result, f.err
}

func TestRebalancePods_NoNodes(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	err := rebalancePods(ctx, client, "default", .25)
	assert.NoError(t, err)
}

func TestRebalancePods_NoReplicaSets(t *testing.T) {
	ctx := context.Background()
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	client := fake.NewSimpleClientset(testNode)

	err := rebalancePods(ctx, client, "default", .25)
	assert.NoError(t, err)
}

func TestRebalancePods_ReplicaSetUnderRollingUpdate(t *testing.T) {
	ctx := context.Background()
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}

	replicas := int32(1)
	testRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rs",
			Namespace: "default",
			UID:       "1234567890",
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "test-job",
					UID:  "abcdefghij",
				},
				{
					Kind: "Deployment",
					Name: "test-job-w",
					UID:  "abcdefghij",
				},
			},
		},
		Status: appsv1.ReplicaSetStatus{
			ObservedGeneration: 1,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
		},
	}
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Job",
					Name: "test-job",
					UID:  "1234567890",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	client := fake.NewSimpleClientset(testNode, testRS, testPod)

	err := rebalancePods(ctx, client, "default", .25)
	assert.NoError(t, err)
}

func TestRebalancePods_PodNotReadyRunning(t *testing.T) {
	ctx := context.Background()
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	testRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-rs",
		},
	}
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "ReplicaSet",
					Name: "test-rs",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}
	client := fake.NewSimpleClientset(testNode, testRS, testPod)

	err := rebalancePods(ctx, client, "default", .25)
	assert.NoError(t, err)
}

func TestRebalancePods_ListNodesError(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	listErr := errors.New("list nodes fail")
	client.PrependReactor("list", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, listErr
	})

	err := rebalancePods(ctx, client, "default", .25)
	if err == nil || !strings.Contains(err.Error(), "failed to list nodes") {
		t.Fatalf("expected wrapped list nodes error, got %v", err)
	}
}

func TestRebalancePods_ListReplicaSetsError(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node"}})
	listErr := errors.New("list rs fail")
	client.PrependReactor("list", "replicasets", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, listErr
	})

	err := rebalancePods(ctx, client, "default", .25)
	if err == nil || !strings.Contains(err.Error(), "failed to get replicasets") {
		t.Fatalf("expected wrapped replicasets error, got %v", err)
	}
}

func TestRebalancePods_ListPodsError(t *testing.T) {
	ctx := context.Background()
	replicas := int32(1)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "default", UID: "uid"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: &replicas},
		Status:     appsv1.ReplicaSetStatus{Replicas: 1},
	}
	client := fake.NewSimpleClientset(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node"}}, rs)
	listErr := errors.New("list pods fail")
	client.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, listErr
	})

	err := rebalancePods(ctx, client, "default", .25)
	if err == nil || !strings.Contains(err.Error(), "failed to list pods") {
		t.Fatalf("expected wrapped pods error, got %v", err)
	}
}

func TestNewCommandInvalidNamespace(t *testing.T) {
	cmd := NewCommand()
	ctx := logger.WithContext(context.Background(), zap.New())
	ctx = client.WithContext(ctx, &client.Options{})
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--namespace", "Invalid_Namespace"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected namespace validation error")
	}
	if !strings.Contains(err.Error(), "invalid namespace") {
		t.Fatalf("expected invalid namespace message, got %v", err)
	}
}

func TestNewCommandClientError(t *testing.T) {
	cmd := NewCommand()
	ctx := logger.WithContext(context.Background(), zap.New())
	ctx = client.WithContext(ctx, &client.Options{})
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--rate", "0.5"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected client creation error")
	}
	if !strings.Contains(err.Error(), "failed to create client") {
		t.Fatalf("expected client creation error message, got %v", err)
	}
}

func TestNewCommandInvalidRate(t *testing.T) {
	cmd := NewCommand()
	ctx := logger.WithContext(context.Background(), zap.New())
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--rate", "1.5"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected rate validation error")
	}
	if !strings.Contains(err.Error(), "rate") {
		t.Fatalf("expected error mentioning rate, got %v", err)
	}
}

func TestRebalancePods_PodNotOwnedByReplicaSet(t *testing.T) {
	ctx := context.Background()
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	testRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rs",
			Namespace: "default",
			UID:       "0123456789",
		},
	}
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Job",
					Name: "test-job",
					UID:  "9876543210",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	client := fake.NewSimpleClientset(testNode, testRS, testPod)

	err := rebalancePods(ctx, client, "default", .25)
	assert.NoError(t, err)
}

func TestRebalancePods_NoNeedToRebalance(t *testing.T) {
	ctx := context.Background()

	node1 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
	node2 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}}

	replicas := int32(2)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs",
			Namespace: "default",
			UID:       "rs-uid",
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
		},
		Status: appsv1.ReplicaSetStatus{
			Replicas: 2,
		},
	}

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "ReplicaSet",
					Name: rs.Name,
					UID:  rs.UID,
				},
			},
		},
		Spec: corev1.PodSpec{NodeName: node1.Name},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "ReplicaSet",
					Name: rs.Name,
					UID:  rs.UID,
				},
			},
		},
		Spec: corev1.PodSpec{NodeName: node2.Name},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	client := fake.NewSimpleClientset(node1, node2, rs, pod1, pod2)
	client.PrependReactor("delete", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		t.Fatalf("unexpected delete action: %#v", action)
		return true, nil, nil
	})

	err := rebalancePods(ctx, client, "default", .25)
	assert.NoError(t, err)
}

func TestRebalancePods_PerformsRebalance(t *testing.T) {
	ctx := context.Background()
	replicas := int32(4)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "heavy-rs",
			Namespace: "default",
			UID:       "heavy",
		},
		Spec:   appsv1.ReplicaSetSpec{Replicas: &replicas},
		Status: appsv1.ReplicaSetStatus{Replicas: 4},
	}
	owner := metav1.OwnerReference{Kind: "ReplicaSet", Name: rs.Name, UID: rs.UID}
	node1 := schedulableNode("node1")
	node2 := schedulableNode("node2")
	pods := []*corev1.Pod{
		readyPod("pod1", node1.Name, owner),
		readyPod("pod2", node1.Name, owner),
		readyPod("pod3", node1.Name, owner),
		readyPod("pod4", node2.Name, owner),
	}
	objects := []runtime.Object{rs, node1, node2}
	for _, p := range pods {
		objects = append(objects, p)
	}
	client := fake.NewSimpleClientset(objects...)

	err := rebalancePods(ctx, client, "default", .5)
	if err != nil {
		t.Fatalf("expected successful rebalance, got %v", err)
	}

	deleteCount := 0
	for _, action := range client.Actions() {
		if action.Matches("delete", "pods") {
			deleteCount++
		}
	}
	if deleteCount == 0 {
		t.Fatalf("expected at least one pod deletion action")
	}
}

func TestRebalancePods_HandlesRebalanceError(t *testing.T) {
	ctx := context.Background()
	replicas := int32(4)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "error-rs",
			Namespace: "default",
			UID:       "error",
		},
		Spec:   appsv1.ReplicaSetSpec{Replicas: &replicas},
		Status: appsv1.ReplicaSetStatus{Replicas: 4},
	}
	owner := metav1.OwnerReference{Kind: "ReplicaSet", Name: rs.Name, UID: rs.UID}
	node1 := schedulableNode("node1")
	node2 := schedulableNode("node2")
	pods := []*corev1.Pod{
		readyPod("pod1", node1.Name, owner),
		readyPod("pod2", node1.Name, owner),
		readyPod("pod3", node1.Name, owner),
		readyPod("pod4", node2.Name, owner),
	}
	objects := []runtime.Object{rs, node1, node2}
	for _, p := range pods {
		objects = append(objects, p)
	}
	client := fake.NewSimpleClientset(objects...)
	client.PrependReactor("delete", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("delete failure")
	})

	err := rebalancePods(ctx, client, "default", .5)
	if err != nil {
		t.Fatalf("expected rebalance to continue despite errors, got %v", err)
	}
}

func TestRebalancePods_LimitsMaxRebalancePerRun(t *testing.T) {
	ctx := context.Background()
	node1 := schedulableNode("node1")
	node2 := schedulableNode("node2")
	client := fake.NewSimpleClientset(node1, node2)
	originalLimit := maxRebalancePerRun
	maxRebalancePerRun = 10
	t.Cleanup(func() { maxRebalancePerRun = originalLimit })

	total := 105
	replicaItems := make([]appsv1.ReplicaSet, total)
	podItems := make([]corev1.Pod, total)
	for i := 0; i < total; i++ {
		name := fmt.Sprintf("rs-%d", i)
		replicas := int32(1)
		replicaItems[i] = appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID(name)},
			Spec:       appsv1.ReplicaSetSpec{Replicas: &replicas},
			Status:     appsv1.ReplicaSetStatus{Replicas: 1},
		}
		owner := metav1.OwnerReference{Kind: "ReplicaSet", Name: name, UID: types.UID(name)}
		podItems[i] = *readyPod(fmt.Sprintf("pod-%d", i), node1.Name, owner)
	}

	client.PrependReactor("list", "replicasets", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, &appsv1.ReplicaSetList{Items: replicaItems}, nil
	})
	client.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, &corev1.PodList{Items: podItems}, nil
	})

	err := rebalancePods(ctx, client, "default", .25)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRebalancePods_SkipsInvalidReplicaState(t *testing.T) {
	ctx := context.Background()
	replicas := int32(1)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "default"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: &replicas},
		Status:     appsv1.ReplicaSetStatus{Replicas: 1},
	}
	client := fake.NewSimpleClientset(schedulableNode("node"), rs)

	originalCandidate := getCandidatePodsFunc
	getCandidatePodsFunc = func(ctx context.Context, c kubernetes.Interface, ns string, nodes []*corev1.Node, replicas []*appsv1.ReplicaSet) ([]*rebalancer.ReplicaState, error) {
		return []*rebalancer.ReplicaState{
			nil,
			{Replicaset: replicas[0]},
		}, nil
	}
	t.Cleanup(func() { getCandidatePodsFunc = originalCandidate })

	originalReb := newRebalancerFunc
	newRebalancerFunc = func(ctx context.Context, current *rebalancer.ReplicaState, rate float32) rebalancerRunner {
		return fakeRebalancer{result: false, err: nil}
	}
	t.Cleanup(func() { newRebalancerFunc = originalReb })

	err := rebalancePods(ctx, client, "default", .25)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestRebalancePods_LimitsRebalanceLoop(t *testing.T) {
	ctx := context.Background()
	replicas := int32(1)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "default"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: &replicas},
		Status:     appsv1.ReplicaSetStatus{Replicas: 1},
	}
	client := fake.NewSimpleClientset(schedulableNode("node"), rs)

	originalLimit := maxRebalancePerRun
	maxRebalancePerRun = 1
	t.Cleanup(func() { maxRebalancePerRun = originalLimit })

	originalCandidate := getCandidatePodsFunc
	getCandidatePodsFunc = func(ctx context.Context, c kubernetes.Interface, ns string, nodes []*corev1.Node, replicas []*appsv1.ReplicaSet) ([]*rebalancer.ReplicaState, error) {
		return []*rebalancer.ReplicaState{
			{Replicaset: replicas[0]},
			{Replicaset: replicas[0]},
		}, nil
	}
	t.Cleanup(func() { getCandidatePodsFunc = originalCandidate })

	var calls int
	originalReb := newRebalancerFunc
	newRebalancerFunc = func(ctx context.Context, current *rebalancer.ReplicaState, rate float32) rebalancerRunner {
		calls++
		return fakeRebalancer{result: false, err: nil}
	}
	t.Cleanup(func() { newRebalancerFunc = originalReb })

	err := rebalancePods(ctx, client, "default", .25)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected rebalancer to be invoked once, got %d", calls)
	}
}

// TestNewCommand verifies default flags and validations
func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	assert.Equal(t, "rebalance-pods", cmd.Use)

	nsFlag := cmd.Flag("namespace")
	if assert.NotNil(t, nsFlag) {
		assert.Equal(t, metav1.NamespaceDefault, nsFlag.DefValue)
		assert.Equal(t, "namespace", nsFlag.Usage)
	}

	rateFlag := cmd.Flag("rate")
	if assert.NotNil(t, rateFlag) {
		assert.Equal(t, "0.25", rateFlag.DefValue)
		assert.Equal(t, "max rebalance rate", rateFlag.Usage)
	}

	// invalid namespace
	err := cmd.Flags().Set("namespace", "invalid#ns")
	assert.NoError(t, err)
	err = cmd.Execute()
	assert.Error(t, err)

	// invalid rate
	cmd = NewCommand()
	err = cmd.Flags().Set("rate", "1.5")
	assert.NoError(t, err)
	err = cmd.Execute()
	assert.Error(t, err)
}

func TestNewCommand_ClientsetError(t *testing.T) {
	missingKubeconfig := filepath.Join(t.TempDir(), "non-existent", "config")
	t.Setenv("KUBECONFIG", missingKubeconfig)

	cmd := NewCommand()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "failed to create client")
	}
}

func TestValidateNamespace(t *testing.T) {
	assert.Error(t, validation.ValidateNamespace(""))
	assert.Error(t, validation.ValidateNamespace("Invalid*"))
	assert.NoError(t, validation.ValidateNamespace("test-ns"))
}

func TestGetTargetReplicaSets(t *testing.T) {
	ctx := context.Background()
	r1rep := int32(2)
	validRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "valid", Namespace: "default"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: &r1rep},
		Status:     appsv1.ReplicaSetStatus{Replicas: 2},
	}
	invalidRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: "default"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: &r1rep},
		Status:     appsv1.ReplicaSetStatus{Replicas: 1},
	}
	client := fake.NewSimpleClientset(validRS, invalidRS)
	rs, err := getTargetReplicaSets(ctx, client, "default")
	assert.NoError(t, err)
	if assert.Len(t, rs, 1) {
		assert.Equal(t, "valid", rs[0].Name)
	}

	client = fake.NewSimpleClientset()
	client.PrependReactor("list", "replicasets", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})
	_, err = getTargetReplicaSets(ctx, client, "default")
	assert.Error(t, err)
}

func TestGetCandidatePods(t *testing.T) {
	ctx := context.Background()
	node1 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
	node2 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}}
	rep := int32(2)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "default", UID: "1"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: &rep},
		Status:     appsv1.ReplicaSetStatus{Replicas: 2},
	}
	goodPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "good", Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{{UID: rs.UID, Kind: "ReplicaSet", Name: rs.Name}},
		},
		Spec:   corev1.PodSpec{NodeName: "node1"},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}},
	}
	notReady := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "notready", Namespace: "default", OwnerReferences: []metav1.OwnerReference{{UID: rs.UID, Kind: "ReplicaSet", Name: rs.Name}}},
		Spec:       corev1.PodSpec{NodeName: "node1"},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	hostPath := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "host", Namespace: "default", OwnerReferences: []metav1.OwnerReference{{UID: rs.UID, Kind: "ReplicaSet", Name: rs.Name}}},
		Spec: corev1.PodSpec{
			NodeName: "node2",
			Volumes:  []corev1.Volume{{Name: "h", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/tmp"}}}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}},
	}
	client := fake.NewSimpleClientset(node1, node2, rs, goodPod, notReady, hostPath)
	nodes := []*corev1.Node{node1, node2}
	rsList := []*appsv1.ReplicaSet{rs}
	ret, err := getCandidatePods(ctx, client, "default", nodes, rsList)
	assert.NoError(t, err)
	if assert.Len(t, ret, 1) {
		assert.Len(t, ret[0].PodStatus, 1)
		assert.Equal(t, "good", ret[0].PodStatus[0].Pod.Name)
		assert.Equal(t, 2, len(ret[0].Nodes))
	}

	client = fake.NewSimpleClientset()
	client.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})
	_, err = getCandidatePods(ctx, client, "default", nodes, rsList)
	assert.Error(t, err)
}

func TestRebalancePods_ErrorCases(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})
	err := rebalancePods(ctx, client, "default", .25)
	assert.Error(t, err)

	client = fake.NewSimpleClientset()
	client.PrependReactor("list", "replicasets", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})
	err = rebalancePods(ctx, client, "default", .25)
	assert.Error(t, err)

	client = fake.NewSimpleClientset()
	client.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})
	err = rebalancePods(ctx, client, "default", .25)
	assert.Error(t, err)
}

func TestRebalancePods_DeleteErrorContinues(t *testing.T) {
	ctx := context.Background()
	replicas := int32(2)

	node1 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
	node2 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}}

	rs1 := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-one",
			Namespace: "default",
			UID:       types.UID("rs-one"),
		},
		Spec:   appsv1.ReplicaSetSpec{Replicas: &replicas},
		Status: appsv1.ReplicaSetStatus{Replicas: 2},
	}
	rs2 := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-two",
			Namespace: "default",
			UID:       types.UID("rs-two"),
		},
		Spec:   appsv1.ReplicaSetSpec{Replicas: &replicas},
		Status: appsv1.ReplicaSetStatus{Replicas: 2},
	}

	pod1a := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1-a",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "ReplicaSet",
				Name: rs1.Name,
				UID:  rs1.UID,
			}},
		},
		Spec: corev1.PodSpec{NodeName: node1.Name},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}
	pod1b := pod1a.DeepCopy()
	pod1b.Name = "pod1-b"

	pod2a := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2-a",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "ReplicaSet",
				Name: rs2.Name,
				UID:  rs2.UID,
			}},
		},
		Spec: corev1.PodSpec{NodeName: node1.Name},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}
	pod2b := pod2a.DeepCopy()
	pod2b.Name = "pod2-b"

	client := fake.NewSimpleClientset(node1, node2, rs1, rs2, pod1a, pod1b, pod2a, pod2b)

	var deleteCalls int
	var successfulDeletes int
	client.PrependReactor("delete", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		deleteCalls++
		if deleteCalls == 1 {
			return true, nil, assert.AnError
		}
		successfulDeletes++
		return true, nil, nil
	})

	err := rebalancePods(ctx, client, "default", .5)
	assert.NoError(t, err)
	assert.Equal(t, 2, deleteCalls)
	assert.Equal(t, 1, successfulDeletes)
}

func TestRebalancePods_LimitReplicaSets(t *testing.T) {
	ctx := context.Background()
	nodes := []*corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "n1"}}, {ObjectMeta: metav1.ObjectMeta{Name: "n2"}}}
	objects := make([]runtime.Object, 0, len(nodes))
	for _, n := range nodes {
		objects = append(objects, n)
	}
	rep := int32(2)
	for i := 0; i < 101; i++ {
		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("rs-%d", i), Namespace: "default", UID: types.UID(fmt.Sprintf("uid-%d", i))},
			Spec:       appsv1.ReplicaSetSpec{Replicas: &rep},
			Status:     appsv1.ReplicaSetStatus{Replicas: 2},
		}
		objects = append(objects, rs)
		for j := 0; j < 2; j++ {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:            fmt.Sprintf("pod-%d-%d", i, j),
					Namespace:       "default",
					OwnerReferences: []metav1.OwnerReference{{UID: rs.UID, Kind: "ReplicaSet", Name: rs.Name}},
				},
				Spec:   corev1.PodSpec{NodeName: "n1"},
				Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}},
			}
			objects = append(objects, pod)
		}
	}
	client := fake.NewSimpleClientset(objects...)
	var deletes int
	client.PrependReactor("delete", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		deletes++
		return true, nil, nil
	})
	err := rebalancePods(ctx, client, "default", .5)
	assert.NoError(t, err)
	assert.Equal(t, 100, deletes)
}
