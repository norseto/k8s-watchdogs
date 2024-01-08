package k8sutils

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ReplicaSetStatus represents the status of a replica set.
type ReplicaSetStatus struct {
	Owners map[types.UID]int
}

// NewReplicaSetStatus returns a new instance of ReplicaSetStatus interface.
// It initializes and returns a rsOwners struct by iterating over the given list of ReplicaSets.
func NewReplicaSetStatus(rs []*appsv1.ReplicaSet) ReplicaSetStatus {
	ret := ReplicaSetStatus{Owners: map[types.UID]int{}}
	for _, r := range rs {
		if r.Spec.Replicas == nil || *r.Spec.Replicas == 0 {
			continue
		}
		for _, o := range r.OwnerReferences {
			ret.Owners[o.UID]++
		}
	}
	return ret
}

// IsRollingUpdating checks if a ReplicaSet is undergoing rolling updates.
// It takes a context and a ReplicaSet as parameters.
// It iterates over the OwnerReferences of the ReplicaSet and checks if any of the OwnerReferences have more than one occurrence in the Owners map.
// If it finds such an OwnerReference, it returns true.
// Otherwise, it returns false.
func (u *ReplicaSetStatus) IsRollingUpdating(_ context.Context, rs *appsv1.ReplicaSet) bool {
	for _, o := range rs.OwnerReferences {
		if u.Owners[o.UID] > 1 {
			return true
		}
	}
	return false
}

// IsPodScheduleLimited checks if the given ReplicaSet has limited scheduling for its Pods.
// It returns true if the scheduling is limited, otherwise false.
func IsPodScheduleLimited(rs appsv1.ReplicaSet) bool {
	podSpec := rs.Spec.Template.Spec
	return podSpec.Affinity != nil || len(podSpec.NodeSelector) > 0
}

// IsPodOwnedBy returns true if the given Pod is owned by the specified ReplicaSet, false otherwise.
// It compares the UID of the ReplicaSet with the UID of each owner reference in the Pod's metadata.
// Example usage:
//
//	rs := &appsv1.ReplicaSet{
//	  ObjectMeta: metav1.ObjectMeta{UID: types.UID("owner-1")},
//	}
//
//	po := &corev1.Pod{
//	  ObjectMeta: metav1.ObjectMeta{
//	    OwnerReferences: []metav1.OwnerReference{
//	      {UID: types.UID("owner-1")},
//	      {UID: types.UID("owner-2")},
//	    },
//	  },
//	}
//
// isOwned := IsPodOwnedBy(rs, po)
// assert.True(t, isOwned)
// po.ObjectMeta.OwnerReferences = []metav1.OwnerReference{{UID: types.UID("owner-3")}}
// isOwned = IsPodOwnedBy(rs, po)
// assert.False(t, isOwned)
func IsPodOwnedBy(rs *appsv1.ReplicaSet, po *corev1.Pod) bool {
	uid := rs.ObjectMeta.UID
	owners := po.ObjectMeta.OwnerReferences
	for _, o := range owners {
		if o.UID == uid {
			return true
		}
	}
	return false
}
