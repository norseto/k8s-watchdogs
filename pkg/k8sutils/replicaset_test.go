package k8sutils

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	orev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
)

func TestNewReplicaSetStatus(t *testing.T) {
	rsList := []appsv1.ReplicaSet{
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

func TestIsPodScheduleLimited(t *testing.T) {
	rs := appsv1.ReplicaSet{
		Spec: appsv1.ReplicaSetSpec{
			Template: orev1.PodTemplateSpec{
				Spec: orev1.PodSpec{
					Affinity: &orev1.Affinity{},
					NodeSelector: map[string]string{
						"key": "value",
					},
				},
			},
		},
	}
	assert.True(t, IsPodScheduleLimited(rs))

	rs = appsv1.ReplicaSet{
		Spec: appsv1.ReplicaSetSpec{
			Template: orev1.PodTemplateSpec{
				Spec: orev1.PodSpec{},
			},
		},
	}
	assert.False(t, IsPodScheduleLimited(rs))
}

func TestIsPodOwnedBy(t *testing.T) {
	rs := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{UID: types.UID("owner-1")},
	}
	po := orev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{UID: types.UID("owner-1")},
				{UID: types.UID("owner-2")},
			},
		},
	}
	assert.True(t, IsPodOwnedBy(rs, po))

	po = orev1.Pod{
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
