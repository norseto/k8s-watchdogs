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

package k8sapps

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
)

func TestNewReplicaSetStatus(t *testing.T) {
	rsList := []*appsv1.ReplicaSet{
		{
			Spec: appsv1.ReplicaSetSpec{Replicas: int32Ptr(2)},
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{UID: types.UID("owner-1")},
					{UID: types.UID("owner-2")},
				},
			},
		},
		{
			Spec: appsv1.ReplicaSetSpec{Replicas: int32Ptr(0)},
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{UID: types.UID("owner-3")},
				},
			},
		},
	}
	expectedOwners := map[types.UID]int{
		types.UID("owner-1"): 1,
		types.UID("owner-2"): 1,
	}

	rsStatus := NewReplicaSetStatus(rsList)

	assert.Equal(t, expectedOwners, rsStatus.Owners)
}

func TestReplicaSetStatus_IsRollingUpdating(t *testing.T) {
	rsStatus := ReplicaSetStatus{
		Owners: map[types.UID]int{
			types.UID("owner-1"): 2,
			types.UID("owner-2"): 1,
		},
	}

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{UID: types.UID("owner-1")},
			},
		},
	}
	assert.True(t, rsStatus.IsRollingUpdating(context.Background(), rs))

	rs = &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{UID: "owner-2"},
			},
		},
	}
	assert.False(t, rsStatus.IsRollingUpdating(context.Background(), rs))

	rs = &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{UID: "owner-3"},
			},
		},
	}
	assert.False(t, rsStatus.IsRollingUpdating(context.Background(), rs))
}

func TestIsPodOwnedBy(t *testing.T) {
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{UID: types.UID("owner-1")},
	}
	po := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{UID: types.UID("owner-1")},
				{UID: types.UID("owner-2")},
			},
		},
	}
	assert.True(t, IsPodOwnedBy(rs, po))

	po = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{UID: "owner-3"},
			},
		},
	}
	assert.False(t, IsPodOwnedBy(rs, po))
}

func int32Ptr(i int32) *int32 {
	return &i
}
