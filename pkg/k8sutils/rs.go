package k8sutils

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Replicaset owners counter.
type rsowners struct {
	owners map[types.UID]int
}

// ReplicaSetStatus represents total ReplicaSets.
type ReplicaSetStatus interface {
	// IsRollingUpdating checks ReplicaSet is now on rollingupdate.
	// Parameters:
	//   rs: ReplicaSet
	// Returns:
	//   True if under rollingupdate.
	IsRollingUpdating(rs *appsv1.ReplicaSet) bool
}

// NewReplicaSetStatus retuns new ReplicasetStatus instance.
// Parameters:
//   rs: Array of ReplicaSet
// Returns:
//   a new instance.
func NewReplicaSetStatus(rs []appsv1.ReplicaSet) ReplicaSetStatus {
	ret := &rsowners{owners: map[types.UID]int{}}
	for _, r := range rs {
		if *r.Spec.Replicas == 0 {
			continue
		}
		for _, o := range r.OwnerReferences {
			ret.owners[o.UID]++
		}
	}
	return ret
}

// IsRollingUpdating checks ReplicaSet is now on rollingupdate.
// Parameters:
//   rs: ReplicaSet
// Returns:
//   True if under rollingupdate.
func (u *rsowners) IsRollingUpdating(rs *appsv1.ReplicaSet) bool {
	for _, o := range rs.OwnerReferences {
		if u.owners[o.UID] > 1 {
			return true
		}
	}
	return false
}

// IsPodScheduleLimeted returns true if Pod Spec of Replicaset has any schedule limeted
// like pod has Affinity, Toleration, or NodeSelector
// Parameter:
//   rs appsv1.ReplicaSet : Target Replicaset
// Returns:
//   bool : True if pod of replicaset scheduling is limited.
func IsPodScheduleLimeted(rs appsv1.ReplicaSet) bool {
	podSpec := rs.Spec.Template.Spec
	return podSpec.Affinity != nil || len(podSpec.Tolerations) > 0 || len(podSpec.NodeSelector) > 0
}

// IsPodOwnedBy determins the owner of the pod is the specified replicaset
// Parameter:
//   rs appsv1.ReplicaSet : Target Replicaset
//   po v1.Pod : Target Pod
// Returns:
//   bool : True if pod is specified replicaset
func IsPodOwnedBy(rs appsv1.ReplicaSet, po v1.Pod) bool {
	uid := rs.ObjectMeta.UID
	owners := po.ObjectMeta.OwnerReferences
	for _, o := range owners {
		if o.UID == uid {
			return true
		}
	}
	return false
}
