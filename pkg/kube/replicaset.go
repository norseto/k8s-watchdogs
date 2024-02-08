/*
MIT License

Copyright (c) 2019 Norihiro Seto

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

package kube

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
	uid := rs.UID
	owners := po.OwnerReferences
	for _, o := range owners {
		if o.UID == uid {
			return true
		}
	}
	return false
}
