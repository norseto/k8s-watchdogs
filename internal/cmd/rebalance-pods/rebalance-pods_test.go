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
	"fmt"
	"testing"

	"github.com/norseto/k8s-watchdogs/internal/pkg/validation"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

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
	t.Setenv("KUBECONFIG", "/non-existent/path")

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
