package rebalancer

import (
	"context"
	"fmt"
	"github.com/norseto/k8s-watchdogs/pkg/k8core"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// ReplicaState represents the state of a replica set.
// It contains a pointer to a replica set object, an array of Nodes,
// and an array of Pod states.
type ReplicaState struct {
	Replicaset *appsv1.ReplicaSet
	Nodes      []*v1.Node
	PodStatus  []*PodStatus
}

// PodStatus represents the state of a Pod.
// It contains a pointer to a Pod object, a pointer to a Node object,
// and a boolean flag indicating whether the Pod has been deleted or not.
type PodStatus struct {
	Pod     *v1.Pod
	Node    *v1.Node
	deleted bool
}

// Rebalancer represents a Rebalancer object.
type Rebalancer struct {
	current          *ReplicaState
	maxRebalanceRate float32
}

// specReplicas returns the number of replicas specified in the current ReplicaSet.
func (r *Rebalancer) specReplicas() int32 {
	if r.current == nil || r.current.Replicaset == nil || r.current.Replicaset.Spec.Replicas == nil {
		return 0
	}
	return *r.current.Replicaset.Spec.Replicas
}

// currentReplicas returns the number of replicas currently running in the ReplicaSet.
func (r *Rebalancer) currentReplicas() int32 {
	if r.current == nil || r.current.Replicaset == nil || r.current.Replicaset.Spec.Replicas == nil {
		return 0
	}
	return r.current.Replicaset.Status.Replicas
}

// filterSchedulables filters the list of scheduleable Nodes based on the Pod specifications.
// It returns a new list of scheduleable Nodes.
// If the current ReplicaState is nil or the length of current.PodStatus is less than 1,
// it returns the original list of Nodes.
// It assigns the first Pod from current.PodStatus to the variable "Pod".
// If "Pod" is nil, it returns the original list of Nodes.
// It calls the k8sutils.FilterScheduleable function to filter the list of Nodes based on the Pod.Spec.
// It assigns the filtered list to the current.Nodes.
func (r *Rebalancer) filterSchedulables() {
	if r.current == nil || len(r.current.PodStatus) < 1 {
		return
	}
	firstPod := r.current.PodStatus[0].Pod
	if firstPod == nil {
		return
	}

	nodes := k8core.FilterScheduleable(r.current.Nodes, &firstPod.Spec)
	r.current.Nodes = nodes
}

// NewRebalancer returns a new instance of the Rebalancer struct with the provided current
// replica state and a default maxRebalanceRate of 0.25.
// The Rebalancer struct contains methods for rebalancing pods across Nodes in a Kubernetes cluster.
func NewRebalancer(current *ReplicaState) *Rebalancer {
	ret := &Rebalancer{current: current, maxRebalanceRate: .25}
	ret.filterSchedulables()
	return ret
}

// Rebalance rebalances the pods across the Nodes in the cluster.
// It returns a boolean indicating if any pods were rebalanced and an error, if any.
// The rebalancing is done by deleting pods from the Node that has the maximum number of pods
// until the Pod count on that Node is less than or equal to the average number of pods across all Nodes plus one.
// The maximum number of pods to be deleted is calculated based on the specified rebalance rate.
// If the number of Nodes is less than 2, the number of replicas is less than 2,
// or the current number of replicas is less than the specified replicas,
// no rebalancing is performed and the function returns false.
func (r *Rebalancer) Rebalance(ctx context.Context, client k8s.Interface) (bool, error) {
	nodeCount := len(r.current.Nodes)
	sr := r.specReplicas()

	if nodeCount < 2 || sr < 2 || r.currentReplicas() < sr {
		return false, nil
	}

	deleted := 0
	maxDel := int(float32(sr) * r.maxRebalanceRate)
	if maxDel < 1 {
		maxDel = 1
	}

	for i := 0; i < maxDel; i++ {
		node, num := r.getNodeWithMaxPods()
		if num < 1 {
			return deleted > 0, nil
		}

		ave := float32(sr) / float32(nodeCount)
		if len(node) <= 0 || float32(num) < ave+1.0 {
			return deleted > 0, nil
		}
		if err := r.deletePodOnNode(ctx, client, node); err != nil {
			return deleted > 0, fmt.Errorf("failed to delete Pod: %v", err)
		}
		deleted++
	}

	return deleted > 0, nil
}

// deletePodOnNode deletes a Pod on specified Node.
func (r *Rebalancer) deletePodOnNode(ctx context.Context, client k8s.Interface, node string) error {
	l := len(r.current.PodStatus)
	for i := 0; i < l; i++ {
		s := r.current.PodStatus[i]
		if s.Node.Name == node && !s.deleted {
			log.Debug("Deleting Pod " + s.Pod.Name + " in " + node)
			s.deleted = true
			return k8core.DeletePod(ctx, client, *s.Pod)
		}
	}
	return nil
}

// getNodeWithMaxPods returns the Node with the maximum number of non-deleted pods and the corresponding Pod count.
func (r *Rebalancer) getNodeWithMaxPods() (string, int) {
	if r.current == nil {
		return "", 0
	}

	podCounts := r.countPodsPerNode()

	maxPods := 0
	nodeNameWithMaxPods := ""
	for nodeName, podCount := range podCounts {
		if podCount > maxPods {
			maxPods = podCount
			nodeNameWithMaxPods = nodeName
		}
	}
	return nodeNameWithMaxPods, maxPods
}

// countPodsPerNode returns a map containing the count of pods per Node in the current replica state.
func (r *Rebalancer) countPodsPerNode() map[string]int {
	podCounts := make(map[string]int)
	for _, s := range r.current.PodStatus {
		if s.deleted {
			continue
		}
		podCounts[s.Node.Name]++
	}
	return podCounts
}
