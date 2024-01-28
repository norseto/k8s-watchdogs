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
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestRebalancePods_NoNodes(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	err := rebalancePods(ctx, client, "default")
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

	err := rebalancePods(ctx, client, "default")
	assert.NoError(t, err)
}

func TestRebalancePods_ReplicaSetUnderRollingUpdate(t *testing.T) {
	ctx := context.Background()
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	testRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-rs",
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
			},
		},
		Status: appsv1.ReplicaSetStatus{
			ObservedGeneration: 1,
		},
	}
	client := fake.NewSimpleClientset(testNode, testRS)

	err := rebalancePods(ctx, client, "default")
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

	err := rebalancePods(ctx, client, "default")
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
			Name: "test-rs",
		},
	}
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Job",
					Name: "test-job",
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

	err := rebalancePods(ctx, client, "default")
	assert.NoError(t, err)
}
